package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime/debug"
	"sort"
	"strings"
	"time"

	"golang.org/x/mod/semver"
)

func (a *app) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		apiMethodNotAllowed(w)
		return
	}
	a.loadModels()

	type statusHarness struct {
		Name      string `json:"name"`
		Available bool   `json:"available"`
		Reason    string `json:"reason,omitempty"`
	}
	type statusWorktree struct {
		Path     string `json:"path"`
		Branch   string `json:"branch"`
		Sessions int    `json:"sessions"`
	}
	type statusProject struct {
		Root      string           `json:"root"`
		Error     string           `json:"error,omitempty"`
		Worktrees []statusWorktree `json:"worktrees"`
	}

	a.mu.Lock()
	roots := strings.FieldsFunc(a.config.GitRootRealDirs, func(r rune) bool { return r == '\n' || r == '\r' })
	pollStatus := a.pollStatus
	if pollStatus == "" {
		pollStatus = "poll.session 0"
	}
	sessionCounts := make(map[string]int, len(a.sessions))
	for folder, sessions := range a.sessions {
		sessionCounts[folder] = len(sessions)
	}
	harnesses := make([]statusHarness, 0, len(a.adapters)+len(a.unavailableAdapters))
	for name := range a.adapters {
		harnesses = append(harnesses, statusHarness{Name: name, Available: true})
	}
	for name, reason := range a.unavailableAdapters {
		harnesses = append(harnesses, statusHarness{Name: name, Available: false, Reason: reason})
	}
	a.mu.Unlock()
	sort.Slice(harnesses, func(i, j int) bool { return harnesses[i].Name < harnesses[j].Name })

	projects := make([]statusProject, 0, len(roots))
	for _, root := range roots {
		project := statusProject{Root: root, Worktrees: []statusWorktree{}}
		dirs, err := gitDirs(root, false)
		if err != nil {
			project.Error = err.Error()
		} else {
			for _, dir := range dirs {
				project.Worktrees = append(project.Worktrees, statusWorktree{
					Path: dir.Dirname, Branch: dir.Branch, Sessions: sessionCounts[dir.Dirname],
				})
			}
		}
		projects = append(projects, project)
	}
	version, notice, updateCheckFailed := checkUpdateNotice()
	apiWriteJSON(w, http.StatusOK, struct {
		Version           string          `json:"version"`
		Update            string          `json:"update"`
		UpdateCommand     string          `json:"update_command"`
		UpdateCheckFailed bool            `json:"update_check_failed"`
		PollStatus        string          `json:"poll_status"`
		Harnesses         []statusHarness `json:"harnesses"`
		Projects          []statusProject `json:"projects"`
	}{Version: version, Update: notice, UpdateCommand: updateCommand, UpdateCheckFailed: updateCheckFailed, PollStatus: pollStatus, Harnesses: harnesses, Projects: projects})
}

const (
	updateCheckURL = "https://proxy.golang.org/github.com/ongspxm/ahweb/@latest"
	updateCommand  = "go install github.com/ongspxm/ahweb@latest"
)

func checkUpdateNotice() (string, string, bool) {
	_formatNotice := func(current, latest string, err error) (string, string, bool) {
		if current == "" {
			current = "unknown"
		}
		if err != nil || !semver.IsValid(current) || !semver.IsValid(latest) {
			return current, "", true
		}
		return current, formatUpdateNotice(current, latest), false
	}

	info, ok := debug.ReadBuildInfo()
	if !ok || info.Main.Version == "" {
		return _formatNotice("unknown", "", fmt.Errorf("missing build version"))
	}
	current := info.Main.Version
	if !semver.IsValid(current) {
		return _formatNotice(current, "", fmt.Errorf("invalid build version"))
	}
	client := http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(updateCheckURL)
	if err != nil {
		return _formatNotice(current, "", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return _formatNotice(current, "", fmt.Errorf("update service returned %s", resp.Status))
	}
	var latest struct {
		Version string `json:"Version"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&latest); err != nil {
		return _formatNotice(current, "", err)
	}
	return _formatNotice(current, latest.Version, nil)
}

func formatUpdateNotice(current, latest string) string {
	if !semver.IsValid(current) || !semver.IsValid(latest) || semver.Compare(latest, current) <= 0 {
		return ""
	}
	return fmt.Sprintf("Update available: %s. Run %s to update.", latest, updateCommand)
}
