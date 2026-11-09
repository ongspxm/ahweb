package main

import (
	"encoding/json"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestGitDirsResolvesNonGitSymlinkToRealPath(t *testing.T) {
	dir := t.TempDir()
	alias := filepath.Join(t.TempDir(), "alias")
	if err := os.Symlink(dir, alias); err != nil {
		t.Fatal(err)
	}

	got, err := gitDirs(alias, false)
	want := []gitWorktree{{Dirname: dir}}
	if err != nil || !slices.Equal(got, want) {
		t.Fatalf("gitDirs(%s) = %q, %v; want %q", alias, got, err, want)
	}
}

func TestGitDirsReturnsRepositoryRootAndWorktreesAndCachesThem(t *testing.T) {
	base, root, runGit := setupGitTestRepo(t)
	worktree := filepath.Join(base, "worktree")
	runGit("worktree", "add", "-q", "-b", "feature", worktree)

	got, err := gitDirs(worktree, false)
	want := []gitWorktree{{Dirname: root, Branch: "master"}, {Dirname: worktree, Branch: "feature"}}
	if err != nil || !slices.Equal(got, want) {
		t.Fatalf("gitDirs(%s) = %q, %v; want %q", worktree, got, err, want)
	}

	gitWorktreeCache.RLock()
	cached := append([]gitWorktree(nil), gitWorktreeCache.byRoot[root]...)
	gitWorktreeCache.RUnlock()
	if !slices.Equal(cached, want) {
		t.Fatalf("gitWorktreeCache[%s] = %q; want %q", root, cached, want)
	}
}

func TestWorktreesHandlerRefreshesListAndCacheAfterWorktreeChanges(t *testing.T) {
	base, root, runGit := setupGitTestRepo(t)
	worktree := filepath.Join(base, "worktree")
	runGit("worktree", "add", "-q", "-b", "feature", worktree)
	odd := filepath.Join(base, "tab\tline\nworktree")
	runGit("worktree", "add", "-q", "-b", "odd", odd)

	check := func(want []string) {
		t.Helper()
		w := testRequest((&app{}).handleWorktrees, "GET", "/api/worktrees?folder="+url.QueryEscape(root), "")
		var response struct {
			Worktrees []gitWorktree `json:"worktrees"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil || w.Code != 200 || len(response.Worktrees) != len(want) {
			t.Fatalf("worktrees response: %d %s; want %q", w.Code, w.Body.String(), want)
		}
		for i, entry := range response.Worktrees {
			branch := map[string]string{root: "master", odd: "odd", worktree: "feature"}[entry.Dirname]
			if entry.Dirname != want[i] || entry.Branch != branch {
				t.Fatalf("worktree %d = %+v", i, entry)
			}
		}
		cached, err := gitDirs(root, false)
		if err != nil || !slices.Equal(cached, response.Worktrees) {
			t.Fatalf("cached worktrees = %q, %v; want %q", cached, err, want)
		}
	}

	check([]string{root, odd, worktree})
	runGit("worktree", "remove", odd)
	check([]string{root, worktree})
}

func setupGitTestRepo(t *testing.T) (base, root string, runGit func(...string)) {
	t.Helper()
	base = t.TempDir()
	root = filepath.Join(base, "repo")
	if err := os.Mkdir(root, 0o755); err != nil {
		t.Fatal(err)
	}
	runGit = func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	runGit("init", "-q", "-b", "master")
	runGit("-c", "user.name=test", "-c", "user.email=test@example.com", "commit", "--allow-empty", "-qm", "init")
	return base, root, runGit
}

func TestWorktreesHandlerLabelsDetachedWorktrees(t *testing.T) {
	base, root, runGit := setupGitTestRepo(t)
	detached := filepath.Join(base, "detached")
	runGit("worktree", "add", "-q", "--detach", detached, "HEAD")

	commit, err := exec.Command("git", "-C", detached, "rev-parse", "HEAD").Output()
	if err != nil {
		t.Fatal(err)
	}
	w := testRequest((&app{}).handleWorktrees, "GET", "/api/worktrees?folder="+url.QueryEscape(root), "")
	var response struct {
		Worktrees []struct {
			Branch  string `json:"branch"`
			Dirname string `json:"dirname"`
		} `json:"worktrees"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil || w.Code != 200 {
		t.Fatalf("worktrees response: %d %s", w.Code, w.Body.String())
	}
	for _, entry := range response.Worktrees {
		if entry.Dirname == detached {
			want := "(detached) " + string(commit[:7])
			if entry.Branch != want {
				t.Fatalf("detached branch = %q; want %q", entry.Branch, want)
			}
			return
		}
	}
	t.Fatalf("detached worktree %q missing from response", detached)
}

func TestDelProjectRemovesConfiguredRootAndTriggersCleanup(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	_, root, _ := setupGitTestRepo(t)
	other := filepath.Join(t.TempDir(), "other")
	a := &app{config: appConfig{GitRootRealDirs: root + "\n" + other}, lastFolderCleanup: time.Now()}
	w := testRequest(a.handleDelProject, "POST", "/api/delproject?folder="+url.QueryEscape(root), "")
	if w.Code != 200 || a.config.GitRootRealDirs != other || !a.lastFolderCleanup.IsZero() {
		t.Fatalf("delete response=%d %s config=%q cleanup=%v", w.Code, w.Body.String(), a.config.GitRootRealDirs, a.lastFolderCleanup)
	}
	path, err := configPath()
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var saved appConfig
	if err := json.Unmarshal(raw, &saved); err != nil || saved.GitRootRealDirs != other {
		t.Fatalf("saved config=%q, err=%v; want %q", saved.GitRootRealDirs, err, other)
	}
}

func TestGitHandlerListsChangesAndReturnsDiff(t *testing.T) {
	_, root, runGit := setupGitTestRepo(t)
	tracked := filepath.Join(root, "tracked.txt")
	if err := os.WriteFile(tracked, []byte("base\n"), 0600); err != nil {
		t.Fatal(err)
	}
	runGit("add", "tracked.txt")
	runGit("-c", "user.name=test", "-c", "user.email=test@example.com", "commit", "-qm", "tracked")
	if err := os.WriteFile(tracked, []byte("staged\n"), 0600); err != nil {
		t.Fatal(err)
	}
	runGit("add", "tracked.txt")
	if err := os.WriteFile(tracked, []byte("working\n"), 0600); err != nil {
		t.Fatal(err)
	}
	oddName := "new\nfile.txt"
	if err := os.WriteFile(filepath.Join(root, oddName), []byte("new"), 0600); err != nil {
		t.Fatal(err)
	}

	q := url.Values{"folder": {root}}
	res := testRequest((&app{}).handleGit, "GET", "/api/git?"+q.Encode(), "")
	var status struct {
		Files []struct {
			Status string `json:"status"`
			Path   string `json:"path"`
		} `json:"files"`
	}
	if res.Code != 200 || json.Unmarshal(res.Body.Bytes(), &status) != nil {
		t.Fatalf("status response: %d %s", res.Code, res.Body.String())
	}
	foundTracked, foundOdd := false, false
	for _, file := range status.Files {
		if file.Path == "tracked.txt" && file.Status == "MM" {
			foundTracked = true
		}
		if file.Path == oddName && file.Status == "??" {
			foundOdd = true
		}
	}
	if !foundTracked || !foundOdd {
		t.Fatalf("status files = %+v", status.Files)
	}

	q.Set("file", "tracked.txt")
	res = testRequest((&app{}).handleGit, "GET", "/api/git?"+q.Encode(), "")
	var diff struct {
		Diff string `json:"diff"`
	}
	if res.Code != 200 || json.Unmarshal(res.Body.Bytes(), &diff) != nil || !strings.Contains(diff.Diff, "-base") || !strings.Contains(diff.Diff, "+working") {
		t.Fatalf("diff response: %d %s", res.Code, res.Body.String())
	}
	q.Set("file", oddName)
	res = testRequest((&app{}).handleGit, "GET", "/api/git?"+q.Encode(), "")
	if res.Code != 200 || json.Unmarshal(res.Body.Bytes(), &diff) != nil || !strings.Contains(diff.Diff, "+new") {
		t.Fatalf("untracked diff response: %d %s", res.Code, res.Body.String())
	}
	if err := os.Remove(tracked); err != nil {
		t.Fatal(err)
	}
	q.Set("file", "tracked.txt")
	res = testRequest((&app{}).handleGit, "GET", "/api/git?"+q.Encode(), "")
	if res.Code != 200 || json.Unmarshal(res.Body.Bytes(), &diff) != nil || !strings.Contains(diff.Diff, "-base") {
		t.Fatalf("deleted file diff response: %d %s", res.Code, res.Body.String())
	}
}

func TestGitHandlerRejectsFileOutsideFolder(t *testing.T) {
	_, root, _ := setupGitTestRepo(t)
	q := url.Values{"folder": {root}, "file": {"../outside"}}
	res := testRequest((&app{}).handleGit, "GET", "/api/git?"+q.Encode(), "")
	if res.Code != 400 {
		t.Fatalf("status = %d, want 400: %s", res.Code, res.Body.String())
	}
}
