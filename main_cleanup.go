package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ongspxm/ahweb/harness"
)

var (
	cleanupMu    sync.Mutex
	cleanupPlans map[string]cleanupPlan
)

type cleanupAction struct {
	folder string
	s      *harness.Session
	last   int64
}

type cleanupPlan struct {
	expires time.Time
	days    int
	actions []cleanupAction
	counts  map[string]int
}

func (a *app) handleCleanup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		apiMethodNotAllowed(w)
		return
	}
	q := r.URL.Query()

	// piping over the request to the single session deletion
	if q.Get("sessid") != "" {
		a.handleDeleteSession(w, r)
		return
	}
	confirm := q.Get("confirm")
	if confirm != "0" && confirm != "1" {
		apiWriteError(w, http.StatusBadRequest, "confirm must be 0 or 1")
		return
	}
	days := 0
	hasDays := false
	if q.Get("days") != "" {
		hasDays = true
		var err error
		days, err = strconv.Atoi(q.Get("days"))
		if err != nil || days < 0 {
			apiWriteError(w, http.StatusBadRequest, "invalid days")
			return
		}
		if days > int((time.Duration(1<<63-1))/(24*time.Hour)) {
			apiWriteError(w, http.StatusBadRequest, "days is too large")
			return
		}
	}
	if confirm == "0" && !hasDays {
		apiWriteError(w, http.StatusBadRequest, "days is required")
		return
	}
	cleanupMu.Lock()
	defer cleanupMu.Unlock()

	// computing the sessions to delete
	var actions []cleanupAction
	var counts map[string]int
	if confirm == "1" {
		planID := q.Get("plan")
		plan, ok := cleanupPlans[planID]
		delete(cleanupPlans, planID)
		if !ok || time.Now().After(plan.expires) || (hasDays && (plan.days != days)) {
			apiWriteError(w, http.StatusConflict, "cleanup preview expired or does not match; preview again")
			return
		}
		actions, counts = plan.actions, plan.counts
	} else {
		a.mu.Lock()
		roots := strings.FieldsFunc(a.config.GitRootRealDirs, func(r rune) bool { return r == '\n' || r == '\r' })
		a.mu.Unlock()
		foldersByPath := make(map[string]bool)
		for _, root := range roots {
			dirs, err := gitDirs(root, true)
			if err != nil {
				apiWriteError(w, http.StatusInternalServerError, fmt.Sprintf("could not list project folders: %v", err))
				return
			}
			for _, dir := range dirs {
				foldersByPath[dir.Dirname] = true
			}
		}
		folders := make([]string, 0, len(foldersByPath))
		for folder := range foldersByPath {
			folders = append(folders, folder)
		}
		sort.Strings(folders)

		cutoff := time.Now().Add(-time.Duration(days) * 24 * time.Hour).UnixMilli()
		counts = make(map[string]int, len(folders))
		for _, folder := range folders {
			if err := a.listFolderSessions(folder); err != nil {
				apiWriteError(w, http.StatusInternalServerError, fmt.Sprintf("could not list sessions in %s: %v", folder, err))
				return
			}
			a.mu.Lock()
			sessions := make([]*harness.Session, 0, len(a.sessions[folder]))
			for _, s := range a.sessions[folder] {
				sessions = append(sessions, s)
			}
			a.mu.Unlock()
			count := 0
			for _, s := range sessions {
				adapter := a.adapters[s.Harness]
				if adapter == nil || s.GetIsBusy() {
					continue
				}
				s.Mu.Lock()
				last := s.LastTimestampMs
				s.Mu.Unlock()
				if last == 0 {
					continue
				}
				if last >= cutoff || !adapter.CanDelete() {
					continue
				}
				count++
				actions = append(actions, cleanupAction{folder: folder, s: s, last: last})
			}
			counts[folder] = count
		}
		var id [24]byte
		if _, err := rand.Read(id[:]); err != nil {
			apiWriteError(w, http.StatusInternalServerError, "could not create cleanup preview")
			return
		}
		planID := hex.EncodeToString(id[:])
		if cleanupPlans == nil {
			cleanupPlans = make(map[string]cleanupPlan)
		}
		now := time.Now()
		for key, plan := range cleanupPlans {
			if now.After(plan.expires) {
				delete(cleanupPlans, key)
			}
		}
		cleanupPlans[planID] = cleanupPlan{expires: now.Add(2 * time.Minute), days: days, actions: actions, counts: counts}
		apiWriteJSON(w, http.StatusOK, map[string]any{"planid": planID, "data": counts})
		return
	}
	result := make(map[string]int, len(counts))
	for folder := range counts {
		result[folder] = 0
	}

	// the logic that asks the poll loop to delete the session
	_deleteSession := func(folder string, s *harness.Session, last int64) error {
		if a.pollJobs == nil {
			return fmt.Errorf("poll workers are not running")
		}
		a.mu.Lock()
		key := sessionKey{harness: s.Harness, id: s.ID}
		stillListed := a.sessions[folder][key] == s
		a.mu.Unlock()
		if !stillListed {
			return fmt.Errorf("session is no longer available")
		}
		result := make(chan error, 1)
		done := &sync.WaitGroup{}
		done.Add(1)
		select {
		case a.pollJobs <- pollJob{session: s, done: done, deleteResult: result, deleteLast: last}:
		case <-r.Context().Done():
			return r.Context().Err()
		}
		err := <-result
		done.Wait()
		if err != nil {
			return err
		}
		a.mu.Lock()
		if a.sessions[folder][key] == s {
			delete(a.sessions[folder], key)
			if len(a.sessions[folder]) == 0 {
				delete(a.sessions, folder)
			}
		}
		a.mu.Unlock()
		a.lastOpened.Lock()
		delete(a.lastOpened.times[folder], s.ID)
		a.lastOpened.Unlock()
		return nil
	}
	var failures []string
	for _, action := range actions {
		if err := _deleteSession(action.folder, action.s, action.last); err != nil {
			failures = append(failures, fmt.Sprintf("%s/%s: %v", action.s.Harness, action.s.ID, err))
			log.Printf("session.cleanup(%s, %s) failed: %v", action.s.Harness, action.s.ID, err)
			continue
		}
		result[action.folder]++
	}
	if len(failures) > 0 {
		apiWriteJSON(w, http.StatusMultiStatus, map[string]any{"data": result, "error": strings.Join(failures, "; ")})
		return
	}
	apiWriteJSON(w, http.StatusOK, map[string]any{"data": result})
}

func (a *app) handleDeleteSession(w http.ResponseWriter, r *http.Request) {
	folder, id := r.URL.Query().Get("folder"), r.URL.Query().Get("sessid")
	if folder == "" || id == "" {
		apiWriteError(w, http.StatusBadRequest, "folder and sessid are required")
		return
	}

	// search for correct session
	a.mu.Lock()
	var s *harness.Session
	for key, candidate := range a.sessions[folder] {
		if key.id != id {
			continue
		}
		if s != nil {
			a.mu.Unlock()
			apiWriteError(w, http.StatusConflict, "sessid is ambiguous; specify a harness")
			return
		}
		s = candidate
	}
	a.mu.Unlock()
	if s == nil {
		apiWriteError(w, http.StatusNotFound, "session not found")
		return
	}

	// send the delete job into the poll loop
	if a.pollJobs == nil {
		apiWriteError(w, http.StatusInternalServerError, "poll workers are not running")
		return
	}
	s.Mu.Lock()
	last := s.LastTimestampMs
	s.Mu.Unlock()
	result := make(chan error, 1)
	done := &sync.WaitGroup{}
	done.Add(1)
	select {
	case a.pollJobs <- pollJob{session: s, done: done, deleteResult: result, deleteLast: last}:
	case <-r.Context().Done():
		return
	}

	if err := <-result; err != nil {
		apiWriteError(w, http.StatusConflict, err.Error())
		return
	}
	done.Wait()

	// do cleanup of the cache
	a.mu.Lock()
	key := sessionKey{harness: s.Harness, id: s.ID}
	if a.sessions[folder][key] == s {
		delete(a.sessions[folder], key)
		if len(a.sessions[folder]) == 0 {
			delete(a.sessions, folder)
		}
	}
	a.mu.Unlock()
	a.lastOpened.Lock()
	delete(a.lastOpened.times[folder], s.ID)
	a.lastOpened.Unlock()
	apiWriteJSON(w, http.StatusOK, map[string]bool{"deleted": true})
}
