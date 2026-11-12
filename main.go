package main

import (
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ongspxm/ahweb/adapters/claude"
	"github.com/ongspxm/ahweb/adapters/codex"
	"github.com/ongspxm/ahweb/adapters/cursor"
	"github.com/ongspxm/ahweb/adapters/opencode"
	"github.com/ongspxm/ahweb/adapters/pi"
	"github.com/ongspxm/ahweb/adapters/prime"
	"github.com/ongspxm/ahweb/harness"
)

// === utils
func apiWriteJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func apiWriteError(w http.ResponseWriter, status int, message string) {
	apiWriteJSON(w, status, map[string]string{"error": message})
}
func apiMethodNotAllowed(w http.ResponseWriter) {
	apiWriteError(w, http.StatusMethodNotAllowed, "method not allowed")
}
func getAbsPath(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("folder is required")
	}
	if value == "~" || strings.HasPrefix(value, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("invalid folder")
		}
		value = filepath.Join(home, strings.TrimPrefix(value, "~/"))
	}
	path, err := filepath.Abs(value)
	if err != nil {
		return "", fmt.Errorf("invalid folder")
	}
	if real, err := filepath.EvalSymlinks(path); err == nil {
		path = real
	}
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return "", fmt.Errorf("folder does not exist")
	}
	return path, nil
}

// === main

//go:embed web/*
var webFiles embed.FS

type sessionKey struct {
	harness string
	id      string
}

type app struct {
	mu sync.Mutex
	// sessions[dirname][harness, sessid]
	sessions               map[string]map[sessionKey]*harness.Session
	cliMu                  sync.Mutex
	cliSessions            map[string]*cliSession
	serverStartTimestampMs int64
	hostIdentity           string
	adapters               map[string]harness.Adapter
	unavailableAdapters    map[string]string
	models                 map[string]map[string]harness.Model
	config                 appConfig
	pollStatus             string
	pollJobs               chan pollJob
	lastFolderCleanup      time.Time
	lastOpened             struct {
		sync.Mutex
		times map[string]map[string]int64
	}
}

const defaultPort = 8321

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	port := flag.Int("port", defaultPort, "HTTP server port")
	flag.Parse()
	if *port < 1 || *port > 65535 {
		log.Fatalf("invalid port %d: must be between 1 and 65535", *port)
	}
	if handled, err := runCommand(flag.Args(), *port); handled {
		if err != nil {
			log.Fatal(err)
		}
		return
	}
	serverHost, err := os.Hostname()
	if err != nil {
		log.Fatalf("host.name failed: %v", err)
	}
	serverUser, err := user.Current()
	if err != nil {
		log.Fatalf("host.user failed: %v", err)
	}
	hostIdentity := strings.ToLower(strings.Join([]string{serverHost, serverUser.Username, strconv.Itoa(*port), runtime.GOOS, runtime.GOARCH}, "_"))
	hostIdentity = strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '_' {
			return r
		}
		return '_'
	}, hostIdentity)
	config, err := loadConfig()
	if err != nil {
		log.Fatalf("config.load failed: %v", err)
	}
	if err := config.validate(); err != nil {
		log.Fatalf("config.validate failed: %v", err)
	}
	server := &app{
		sessions:               make(map[string]map[sessionKey]*harness.Session),
		serverStartTimestampMs: time.Now().UnixMilli(),
		hostIdentity:           hostIdentity,
		adapters: map[string]harness.Adapter{
			"pi": &pi.Adapter{}, "prime": &prime.Adapter{}, "opencode": &opencode.Adapter{}, "claude": &claude.Adapter{}, "codex": &codex.Adapter{}, "cursor": &cursor.Adapter{},
		},
		models:              make(map[string]map[string]harness.Model),
		unavailableAdapters: make(map[string]string),
		config:              config, lastFolderCleanup: time.Now()}
	for name, adp := range server.adapters {
		if err := adp.Ready(); err != nil {
			log.Printf("harness.ready(%s) failed: %v", name, err)
			server.unavailableAdapters[name] = err.Error()
			delete(server.adapters, name)
		}
	}
	server.loadModels()
	server.lastOpened.times = make(map[string]map[string]int64)
	seenRoots := make(map[string]bool)
	for _, value := range strings.Split(server.config.GitRootRealDirs, "\n") {
		root, err := getAbsPath(value)
		if err != nil || seenRoots[root] {
			continue
		}
		seenRoots[root] = true
		if err := server.loadProject(root); err != nil {
			log.Printf("git.worktrees(%s) failed: %v", root, err)
		}
	}
	server.pollLoop()

	// all the server stuff
	assets, err := fs.Sub(webFiles, "web")
	if err != nil {
		log.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.FS(assets)))

	mux.HandleFunc("/host", server.handleHost)
	mux.HandleFunc("/api/status", server.handleStatus)
	mux.HandleFunc("/api/config", server.handleConfig)
	mux.HandleFunc("/api/models", server.handleModels)
	mux.HandleFunc("/api/file", server.handleFile)
	mux.HandleFunc("/api/addproject", server.handleAddProject)
	mux.HandleFunc("/api/delproject", server.handleDelProject)
	mux.HandleFunc("/api/worktrees", server.handleWorktrees)
	mux.HandleFunc("/api/git", server.handleGit)
	mux.HandleFunc("/api/list", server.handleList)
	mux.HandleFunc("/cleanup", server.handleCleanup)
	mux.HandleFunc("/api/events", server.handleEvents)
	mux.HandleFunc("/api/stop", server.handleStop)
	mux.HandleFunc("/api/unqueue", server.handleUnqueue)
	mux.HandleFunc("/api/prompt", server.handlePrompt)
	mux.HandleFunc("/api/cli", server.handleCLI)
	address := fmt.Sprintf("0.0.0.0:%d", *port)
	log.Printf("ahweb listening on http://%s", address)
	if err := http.ListenAndServe(address, mux); err != nil {
		log.Fatal(err)
	}
}

// === api fns
func (a *app) handleHost(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		apiMethodNotAllowed(w)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = fmt.Fprint(w, a.hostIdentity)

}

func (a *app) handleConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		apiMethodNotAllowed(w)
		return
	}
	key := strings.TrimSpace(r.URL.Query().Get("key"))
	_config2map := func(c appConfig) (map[string]any, error) {
		data, err := json.Marshal(c)
		if err != nil {
			return nil, err
		}
		var values map[string]any
		if err := json.Unmarshal(data, &values); err != nil {
			return nil, err
		}
		return values, nil
	}
	a.mu.Lock()
	if r.Method == http.MethodGet {
		config := a.config
		a.mu.Unlock()
		if key == "" {
			apiWriteJSON(w, http.StatusOK, config)
			return
		}
		values, err := _config2map(config)
		if err != nil {
			apiWriteError(w, http.StatusInternalServerError, "could not read config")
			return
		}
		if value, ok := values[key]; ok {
			apiWriteJSON(w, http.StatusOK, value)
		} else {
			apiWriteError(w, http.StatusBadRequest, "unknown config")
		}
		return
	}
	updated := a.config
	setters := map[string]func(*appConfig, string) error{
		"git_root_real_dirs": func(c *appConfig, value string) error {
			roots, seen := []string{}, map[string]bool{}
			for _, path := range strings.FieldsFunc(value, func(r rune) bool { return r == '\n' || r == '\r' }) {
				root, err := getAbsPath(path)
				if err != nil {
					return err
				}
				dirs, err := gitDirs(root, false)
				if err != nil {
					return err
				}
				root = dirs[0].Dirname
				if !seen[root] {
					roots, seen[root] = append(roots, root), true
				}
			}
			c.GitRootRealDirs = strings.Join(roots, "\n")
			return nil
		},
		"num_empty_poll_before_cli_reset": func(c *appConfig, value string) error {
			n, err := strconv.Atoi(value)
			if err != nil || n < 1 {
				return fmt.Errorf("invalid integer config")
			}
			c.NumEmptyPollBeforeCLIReset = n
			return nil
		},
		"session_cleanup_after_minutes": func(c *appConfig, value string) error {
			n, err := strconv.Atoi(value)
			if err != nil {
				return fmt.Errorf("invalid integer config")
			}
			c.SessionCleanupAfterMinutes = n
			return nil
		},
		"folder_cleanup_after_minutes": func(c *appConfig, value string) error {
			n, err := strconv.Atoi(value)
			if err != nil || n < 1 {
				return fmt.Errorf("invalid integer config")
			}
			c.FolderCleanupAfterMinutes = n
			return nil
		},
		"max_image_MB": func(c *appConfig, value string) error {
			n, err := strconv.Atoi(value)
			if err != nil {
				return fmt.Errorf("invalid integer config")
			}
			c.MaxImageMB = n
			return nil
		},
		"max_prompt_images": func(c *appConfig, value string) error {
			n, err := strconv.Atoi(value)
			if err != nil {
				return fmt.Errorf("invalid integer config")
			}
			c.MaxPromptImages = n
			return nil
		},
		"tour_done": func(c *appConfig, value string) error {
			value = strings.TrimSpace(value)
			if value == "" {
				return fmt.Errorf("tour ID is required")
			}
			for _, done := range c.ToursDone {
				if done == value {
					return nil
				}
			}
			c.ToursDone = append(c.ToursDone, value)
			return nil
		},
		"pi_default_model":           func(c *appConfig, value string) error { c.PiDefaultModel = value; return nil },
		"pi_default_reasoning":       func(c *appConfig, value string) error { c.PiDefaultReasoning = value; return nil },
		"prime_default_model":        func(c *appConfig, value string) error { c.PrimeDefaultModel = value; return nil },
		"prime_default_reasoning":    func(c *appConfig, value string) error { c.PrimeDefaultReasoning = value; return nil },
		"opencode_default_model":     func(c *appConfig, value string) error { c.OpencodeDefaultModel = value; return nil },
		"opencode_default_reasoning": func(c *appConfig, value string) error { c.OpencodeDefaultReasoning = value; return nil },
		"claude_default_model":       func(c *appConfig, value string) error { c.ClaudeDefaultModel = value; return nil },
		"claude_default_reasoning":   func(c *appConfig, value string) error { c.ClaudeDefaultReasoning = value; return nil },
		"codex_default_model":        func(c *appConfig, value string) error { c.CodexDefaultModel = value; return nil },
		"codex_default_reasoning":    func(c *appConfig, value string) error { c.CodexDefaultReasoning = value; return nil },
		"cursor_default_model":       func(c *appConfig, value string) error { c.CursorDefaultModel = value; return nil },
		"cursor_default_reasoning":   func(c *appConfig, value string) error { c.CursorDefaultReasoning = value; return nil },
	}
	set, ok := setters[key]
	if !ok {
		a.mu.Unlock()
		apiWriteError(w, http.StatusBadRequest, "unknown or read-only config")
		return
	}
	if err := set(&updated, r.URL.Query().Get("value")); err != nil {
		a.mu.Unlock()
		apiWriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := updated.validate(); err != nil {
		a.mu.Unlock()
		apiWriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := saveConfig(updated); err != nil {
		a.mu.Unlock()
		log.Printf("config.save failed: key=%s error=%v", key, err)
		apiWriteError(w, http.StatusInternalServerError, "could not save config")
		return
	}
	a.config = updated
	a.mu.Unlock()
	if key == "git_root_real_dirs" {
		for _, root := range strings.Split(updated.GitRootRealDirs, "\n") {
			if root != "" {
				if err := a.loadProject(root); err != nil {
					log.Printf("git.worktrees(%s) failed: %v", root, err)
				}
			}
		}
	}
	values, err := _config2map(updated)
	if err != nil {
		apiWriteError(w, http.StatusInternalServerError, "could not read config")
		return
	}
	if key == "tour_done" {
		apiWriteJSON(w, http.StatusOK, values["tours_done"])
		return
	}
	apiWriteJSON(w, http.StatusOK, values[key])

}
