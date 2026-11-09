package main

import (
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"
)

// === git api
type gitWorktree struct {
	Dirname string `json:"dirname"`
	Branch  string `json:"branch"`
}

var gitWorktreeCache = struct {
	sync.RWMutex
	byRoot map[string][]gitWorktree
}{byRoot: make(map[string][]gitWorktree)}

func gitDirs(folder string, refresh bool) ([]gitWorktree, error) {
	folder, err := getAbsPath(folder)
	if err != nil {
		return nil, err
	}
	if !refresh {
		gitWorktreeCache.RLock()
		for _, dirs := range gitWorktreeCache.byRoot {
			for _, dir := range dirs {
				if dir.Dirname == folder {
					cached := append([]gitWorktree(nil), dirs...)
					gitWorktreeCache.RUnlock()
					return cached, nil
				}
			}
		}
		gitWorktreeCache.RUnlock()
	}

	if _, err := exec.Command("git", "-C", folder, "rev-parse", "--show-toplevel").Output(); err != nil {
		return []gitWorktree{{Dirname: folder}}, nil
	}
	out, err := exec.Command("git", "-C", folder, "worktree", "list", "--porcelain", "-z").Output()
	if err != nil {
		return nil, fmt.Errorf("could not list git worktrees")
	}
	var dirs []gitWorktree
	for _, line := range strings.Split(string(out), "\x00") {
		if dir, ok := strings.CutPrefix(line, "worktree "); ok {
			real, err := getAbsPath(dir)
			if err != nil {
				return nil, err
			}
			dirs = append(dirs, gitWorktree{Dirname: real})
		} else if len(dirs) > 0 {
			entry := &dirs[len(dirs)-1]
			if branch, ok := strings.CutPrefix(line, "branch refs/heads/"); ok {
				entry.Branch = branch
			} else if hash, ok := strings.CutPrefix(line, "HEAD "); ok && len(hash) >= 7 {
				entry.Branch = "(detached) " + hash[:7]
			}
		}
	}
	if len(dirs) == 0 {
		return nil, fmt.Errorf("git repository has no worktree")
	}
	gitWorktreeCache.Lock()
	gitWorktreeCache.byRoot[dirs[0].Dirname] = append([]gitWorktree(nil), dirs...)
	gitWorktreeCache.Unlock()
	return dirs, nil
}

func (a *app) loadProject(root string) error {
	dirs, err := gitDirs(root, false)
	if err != nil {
		return err
	}
	for _, dir := range dirs {
		a.listFolderSessions(dir.Dirname)
	}
	return nil
}

func (a *app) handleAddProject(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		apiMethodNotAllowed(w)
		return
	}
	dirs, err := gitDirs(r.URL.Query().Get("folder"), false)
	if err != nil {
		apiWriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	root := dirs[0].Dirname
	a.mu.Lock()
	roots := strings.FieldsFunc(a.config.GitRootRealDirs, func(r rune) bool { return r == '\n' || r == '\r' })
	if !slices.Contains(roots, root) {
		roots = append(roots, root)
	}
	updated := a.config
	updated.GitRootRealDirs = strings.Join(roots, "\n")
	if err := saveConfig(updated); err != nil {
		a.mu.Unlock()
		log.Printf("config.save failed: key=git_root_real_dirs error=%v", err)
		apiWriteError(w, http.StatusInternalServerError, "could not save config")
		return
	}
	a.config = updated
	a.mu.Unlock()
	if err := a.loadProject(root); err != nil {
		log.Printf("git.worktrees(%s) failed: %v", root, err)
	}
	apiWriteJSON(w, http.StatusOK, map[string]string{"folder": root})
}

func (a *app) handleDelProject(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		apiMethodNotAllowed(w)
		return
	}
	folder := strings.TrimSpace(r.URL.Query().Get("folder"))
	if folder == "" {
		apiWriteError(w, http.StatusBadRequest, "folder is required")
		return
	}
	a.mu.Lock()
	roots := strings.FieldsFunc(a.config.GitRootRealDirs, func(r rune) bool { return r == '\n' || r == '\r' })
	filtered := roots[:0]
	found := false
	for _, root := range roots {
		if root == folder {
			found = true
			continue
		}
		filtered = append(filtered, root)
	}
	if !found {
		a.mu.Unlock()
		apiWriteError(w, http.StatusNotFound, "project not found")
		return
	}
	updated := a.config
	updated.GitRootRealDirs = strings.Join(filtered, "\n")
	if err := saveConfig(updated); err != nil {
		a.mu.Unlock()
		log.Printf("config.save failed: key=git_root_real_dirs error=%v", err)
		apiWriteError(w, http.StatusInternalServerError, "could not save config")
		return
	}
	a.config = updated
	a.lastFolderCleanup = time.Time{}
	a.mu.Unlock()
	apiWriteJSON(w, http.StatusOK, map[string]string{"folder": folder})
}

func (a *app) handleWorktrees(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		apiMethodNotAllowed(w)
		return
	}
	requested, err := getAbsPath(r.URL.Query().Get("folder"))
	if err != nil {
		apiWriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	dirs, err := gitDirs(requested, true)
	if err != nil || dirs[0].Dirname != requested {
		apiWriteError(w, http.StatusBadRequest, "folder must be a git root")
		return
	}
	apiWriteJSON(w, http.StatusOK, map[string]any{"folder": requested, "worktrees": dirs})
}

func (a *app) handleGit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		apiMethodNotAllowed(w)
		return
	}
	folder, err := getAbsPath(r.URL.Query().Get("folder"))
	if err != nil {
		apiWriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	if file := r.URL.Query().Get("file"); file != "" {
		path := filepath.Clean(filepath.FromSlash(file))
		if filepath.IsAbs(path) || path == "." || path == ".." || strings.HasPrefix(path, ".."+string(filepath.Separator)) {
			apiWriteError(w, http.StatusBadRequest, "file must be relative to folder")
			return
		}
		literalPath := ":(literal)" + filepath.ToSlash(path)
		out, err := exec.Command("git", "-C", folder, "diff", "HEAD", "--", literalPath).Output()
		if err == nil && len(out) == 0 {
			untracked, statusErr := exec.Command("git", "-C", folder, "ls-files", "--others", "--exclude-standard", "-z", "--", literalPath).Output()
			if statusErr == nil && len(untracked) > 0 {
				fullPath := filepath.Join(folder, path)
				out, err = exec.Command("git", "-C", folder, "diff", "--no-index", "/dev/null", fullPath).Output()
				if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
					err = nil
				}
			}
		}
		if err != nil {
			apiWriteError(w, http.StatusBadRequest, "could not read git diff")
			return
		}
		apiWriteJSON(w, http.StatusOK, map[string]string{"diff": string(out)})
		return
	}

	out, err := exec.Command("git", "-C", folder, "status", "--porcelain", "-z", "--untracked-files=all").Output()
	if err != nil {
		apiWriteError(w, http.StatusBadRequest, "could not read git status")
		return
	}
	type gitFile struct {
		Status string `json:"status"`
		Path   string `json:"path"`
	}
	files := []gitFile{}
	entries := strings.Split(string(out), "\x00")
	for i := 0; i < len(entries); i++ {
		entry := entries[i]
		if entry == "" {
			continue
		}
		if len(entry) < 4 {
			apiWriteError(w, http.StatusInternalServerError, "invalid git status output")
			return
		}
		status, path := entry[:2], entry[3:]
		files = append(files, gitFile{Status: status, Path: path})
		if strings.ContainsAny(status, "RC") && i+1 < len(entries) {
			i++
		}
	}
	apiWriteJSON(w, http.StatusOK, map[string]any{"files": files})
}
