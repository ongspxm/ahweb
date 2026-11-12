package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ongspxm/ahweb/harness"
)

func (a *app) lastOpenedAt(realFolder, sessID string) int64 {
	a.lastOpened.Lock()
	defer a.lastOpened.Unlock()
	if a.lastOpened.times == nil {
		a.lastOpened.times = make(map[string]map[string]int64)
	}
	if a.lastOpened.times[realFolder] == nil {
		a.lastOpened.times[realFolder] = make(map[string]int64)
	}
	if a.lastOpened.times[realFolder][sessID] == 0 {
		a.lastOpened.times[realFolder][sessID] = a.serverStartTimestampMs
	}
	return a.lastOpened.times[realFolder][sessID]
}

func (a *app) setLastOpened(realFolder, sessID string) {
	a.lastOpened.Lock()
	defer a.lastOpened.Unlock()
	if a.lastOpened.times == nil {
		a.lastOpened.times = make(map[string]map[string]int64)
	}
	if a.lastOpened.times[realFolder] == nil {
		a.lastOpened.times[realFolder] = make(map[string]int64)
	}
	a.lastOpened.times[realFolder][sessID] = time.Now().UnixMilli()
}

// utils
func (a *app) sessionLocked(folder, harnessName, sessID string) *harness.Session {
	// this has to be called when the a.mu is locked
	return a.sessions[folder][sessionKey{harness: harnessName, id: sessID}]
}
func (a *app) listFolderSessions(folder string) error {
	var listErrors []error
	for name, adapter := range a.adapters {
		listed, err := adapter.ListSessions(folder)
		if err != nil {
			log.Printf("adapter.list(%s, %s) failed: %v", name, folder, err)
			listErrors = append(listErrors, fmt.Errorf("%s: %w", name, err))
			continue
		}
		a.mu.Lock()
		if a.sessions[folder] == nil {
			a.sessions[folder] = make(map[sessionKey]*harness.Session)
		}
		for _, s := range listed {
			key := sessionKey{harness: s.Harness, id: s.ID}
			if existing := a.sessions[folder][key]; existing != nil {
				// Keep identity fields immutable; poll workers read them without s.Mu.
				existing.Mu.Lock()
				if existing.IsDeleting || existing.IsDeleted {
					existing.Mu.Unlock()
					continue
				}
				existing.Name = s.Name
				existing.FirstUserMsg = s.FirstUserMsg
				existing.LastTimestampMs = s.LastTimestampMs
				existing.ParentSessID = s.ParentSessID
				existing.Mu.Unlock()
			} else {
				a.sessions[folder][key] = &harness.Session{SessionLite: s}
			}
		}
		a.mu.Unlock()
	}
	return errors.Join(listErrors...)
}

func sessionName(s harness.SessionLite) string {
	if s.Name != "" {
		return s.Name
	}
	name := []rune(strings.TrimSpace(s.FirstUserMsg))
	if len(name) > 140 {
		name = name[:140]
	}
	return string(name)
}

// handle fns
func (a *app) handleList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		apiMethodNotAllowed(w)
		return
	}
	realFolder, err := getAbsPath(r.URL.Query().Get("folder"))
	if err != nil {
		apiWriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	a.listFolderSessions(realFolder)

	type apiSession struct {
		SessID        string `json:"sessid"`
		Name          string `json:"name"`
		LastTimestamp int64  `json:"last_timestamp"`
		LastOpened    int64  `json:"last_opened"`
		Harness       string `json:"harness"`
		Busy          bool   `json:"busy"`
		ParentSessID  string `json:"parent_sessid,omitempty"`
	}
	response := struct {
		Folder   string       `json:"folder"`
		Sessions []apiSession `json:"sessions"`
	}{Folder: realFolder}
	a.mu.Lock()
	for _, value := range a.sessions[realFolder] {
		s := value
		busy := s.GetIsBusy()
		s.Mu.Lock()
		lite := s.SessionLite
		s.Mu.Unlock()
		response.Sessions = append(response.Sessions, apiSession{
			SessID: lite.ID, Name: sessionName(lite), Harness: lite.Harness, Busy: busy, LastTimestamp: lite.LastTimestampMs, LastOpened: a.lastOpenedAt(realFolder, lite.ID), ParentSessID: lite.ParentSessID})
	}
	a.mu.Unlock()
	sort.Slice(response.Sessions, func(i, j int) bool {
		a, b := response.Sessions[i].LastTimestamp, response.Sessions[j].LastTimestamp
		if a == 0 || b == 0 {
			return a == 0 && b != 0
		}
		return a > b
	})
	apiWriteJSON(w, http.StatusOK, response)
}

func (a *app) handleEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		apiMethodNotAllowed(w)
		return
	}
	folder, err := getAbsPath(r.URL.Query().Get("folder"))
	if err != nil {
		apiWriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	sessID := strings.TrimSpace(r.URL.Query().Get("sessid"))
	if sessID == "" {
		apiWriteError(w, http.StatusBadRequest, "sessid is required")
		return
	}
	harnessName := strings.TrimSpace(r.URL.Query().Get("harness"))
	if harnessName == "" {
		apiWriteError(w, http.StatusBadRequest, "harness is required")
		return
	}
	startFrom := int64(0)
	if raw := strings.TrimSpace(r.URL.Query().Get("start_from")); raw != "" {
		startFrom, err = strconv.ParseInt(raw, 10, 64)
		if err != nil || startFrom < -1 {
			apiWriteError(w, http.StatusBadRequest, "invalid start_from")
			return
		}
	}

	opened, err := strconv.ParseBool(r.URL.Query().Get("opened"))
	if err != nil && r.URL.Query().Get("opened") != "" {
		apiWriteError(w, http.StatusBadRequest, "invalid opened")
		return
	}
	a.mu.Lock()
	s := a.sessionLocked(folder, harnessName, sessID)
	a.mu.Unlock()
	if s == nil {
		apiWriteError(w, http.StatusNotFound, "session not found")
		return
	}
	if opened {
		a.setLastOpened(folder, sessID)
	}
	lastOpened := a.lastOpenedAt(folder, sessID)

	if startFrom == -1 {
		s.Mu.Lock()
		s.DoFullHistory = true
		if s.DoneFullHistory == nil {
			s.DoneFullHistory = make(chan struct{})
		}
		done := s.DoneFullHistory
		s.Mu.Unlock()
		select {
		case <-done:
		case <-r.Context().Done():
			return
		}
	}

	busy := s.GetIsBusy()
	s.Mu.Lock()
	tstamp := s.LastTimestampMs
	type apiEvent struct {
		harness.Event
		Seq int64 `json:"seq"`
	}
	events := make([]apiEvent, 0, len(s.Events))
	for i, event := range s.Events {
		if int64(i) >= startFrom {
			events = append(events, apiEvent{Event: event, Seq: int64(i)})
		}
	}
	queued := s.QueuedMsg
	queuedImages := s.QueuedImages
	name := sessionName(s.SessionLite)
	stats := map[string]any{"full_input_tokens": s.FullInputToken, "full_output_tokens": s.FullOutputToken, "cache_read_tokens": s.CacheReadToken, "cache_write_tokens": s.CacheWriteToken, "full_cost": s.FullCost, "context_tokens": s.ContextTokens, "context_window": s.ContextWindow}
	s.Mu.Unlock()

	apiWriteJSON(w, http.StatusOK, map[string]any{"sessid": sessID, "name": name, "events": events, "busy": busy, "queued": queued, "queued_images": queuedImages, "stats": stats, "tstamp": tstamp, "last_opened": lastOpened})
}

func (a *app) handleStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		apiMethodNotAllowed(w)
		return
	}
	folder, err := getAbsPath(r.URL.Query().Get("folder"))
	if err != nil {
		apiWriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	sessID := strings.TrimSpace(r.URL.Query().Get("sessid"))
	if sessID == "" {
		apiWriteError(w, http.StatusBadRequest, "sessid is required")
		return
	}
	harnessName := strings.TrimSpace(r.URL.Query().Get("harness"))
	if harnessName == "" {
		apiWriteError(w, http.StatusBadRequest, "harness is required")
		return
	}
	a.mu.Lock()
	s := a.sessionLocked(folder, harnessName, sessID)
	a.mu.Unlock()
	if s == nil {
		apiWriteError(w, http.StatusNotFound, "session not found")
		return
	}
	s.Mu.Lock()
	if s.IsDeleted || s.IsDeleting || s.IsStopping {
		s.Mu.Unlock()
		apiWriteError(w, http.StatusConflict, "session is being deleted or stopped")
		return
	}
	s.IsStopping = true
	s.Mu.Unlock()
	if err := a.adapters[s.Harness].Stop(s); err != nil {
		log.Printf("session.stop failed: sessid=%s error=%v", sessID, err)
		s.OpMu.Lock()
		_ = a.adapters[s.Harness].CleanUp(s)
		s.OpMu.Unlock()
		s.Mu.Lock()
		s.IsStopping = false
		s.Mu.Unlock()
		apiWriteError(w, http.StatusBadGateway, err.Error())
		return
	}
	s.Mu.Lock()
	s.IsStopping = false
	queued := s.SendingMsg
	if queued != "" && s.QueuedMsg != "" {
		queued += "\n"
	}
	queued += s.QueuedMsg
	queuedImages := append(append([]string{}, s.SendingImages...), s.QueuedImages...)
	s.QueuedMsg = ""
	s.QueuedImages = nil
	s.SendingMsg, s.SendingImages = "", nil
	s.BusyMsg, s.BusyCli = false, false
	s.Mu.Unlock()
	apiWriteJSON(w, http.StatusOK, map[string]any{"sessid": sessID, "busy": s.GetIsBusy(), "queued": queued, "queued_images": queuedImages})
}

func (a *app) handleUnqueue(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		apiMethodNotAllowed(w)
		return
	}
	folder, err := getAbsPath(r.URL.Query().Get("folder"))
	if err != nil {
		apiWriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	sessID := strings.TrimSpace(r.URL.Query().Get("sessid"))
	if sessID == "" {
		apiWriteError(w, http.StatusBadRequest, "sessid is required")
		return
	}
	harnessName := strings.TrimSpace(r.URL.Query().Get("harness"))
	if harnessName == "" {
		apiWriteError(w, http.StatusBadRequest, "harness is required")
		return
	}
	a.mu.Lock()
	s := a.sessionLocked(folder, harnessName, sessID)
	a.mu.Unlock()
	if s == nil {
		apiWriteError(w, http.StatusNotFound, "session not found")
		return
	}
	s.Mu.Lock()
	if s.IsDeleted || s.IsDeleting || s.IsStopping {
		s.Mu.Unlock()
		apiWriteError(w, http.StatusConflict, "session is being deleted or stopped")
		return
	}
	queued, queuedImages := s.QueuedMsg, append([]string{}, s.QueuedImages...)
	s.QueuedMsg, s.QueuedImages = "", nil
	s.Mu.Unlock()
	apiWriteJSON(w, http.StatusOK, map[string]any{"sessid": sessID, "busy": s.GetIsBusy(), "queued": queued, "queued_images": queuedImages})
}

func (a *app) handlePrompt(w http.ResponseWriter, r *http.Request) {
	// handling api params
	if r.Method != http.MethodPost {
		apiMethodNotAllowed(w)
		return
	}
	var req struct {
		Folder    string   `json:"folder"`
		SessID    string   `json:"sessid"`
		Prompt    string   `json:"prompt"`
		Harness   string   `json:"harness"`
		Model     string   `json:"model"`
		Reasoning string   `json:"reasoning"`
		Images    []string `json:"images"`
	}
	a.mu.Lock()
	maxImageBytes, maxPromptImages := int64(a.config.MaxImageMB)*(1<<20), a.config.MaxPromptImages
	a.mu.Unlock()
	limit := int64(maxPromptImages)*(maxImageBytes*4/3+256) + 1<<20
	r.Body = http.MaxBytesReader(w, r.Body, limit)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			apiWriteError(w, http.StatusRequestEntityTooLarge, "request too large")
		} else {
			apiWriteError(w, http.StatusBadRequest, "invalid request body")
		}
		return
	}
	req.Harness = strings.TrimSpace(req.Harness)
	if req.Harness == "" {
		apiWriteError(w, http.StatusBadRequest, "harness is required")
		return
	}
	if len(req.Images) > maxPromptImages {
		apiWriteError(w, http.StatusBadRequest, "too many images")
		return
	}
	for _, uri := range req.Images {
		// the parsing here is just for validation
		img, err := harness.ParseImage(uri)
		if err != nil {
			apiWriteError(w, http.StatusBadRequest, "invalid image")
			return
		}
		data, err := base64.StdEncoding.DecodeString(img.Data)
		if err != nil {
			apiWriteError(w, http.StatusBadRequest, "invalid image data")
			return
		}
		if int64(len(data)) > maxImageBytes {
			apiWriteError(w, http.StatusRequestEntityTooLarge, "image too large")
			return
		}
	}
	folder, err := getAbsPath(req.Folder)
	if err != nil {
		apiWriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	if strings.TrimSpace(req.Prompt) == "" {
		apiWriteError(w, http.StatusBadRequest, "prompt is required")
		return
	}
	// get correct sess; create one when no session is selected.
	harnessName := req.Harness
	a.mu.Lock()
	if a.adapters[harnessName] == nil {
		a.mu.Unlock()
		apiWriteError(w, http.StatusBadRequest, "unsupported harness")
		return
	}
	var s *harness.Session
	if strings.TrimSpace(req.SessID) != "" {
		s = a.sessionLocked(folder, harnessName, req.SessID)
	} else {
		idBytes := make([]byte, 16)
		if _, err := rand.Read(idBytes); err != nil {
			a.mu.Unlock()
			apiWriteError(w, http.StatusInternalServerError, "could not create session")
			return
		}
		id := hex.EncodeToString(idBytes)
		s = &harness.Session{SessionLite: harness.SessionLite{ID: id, RealPath: folder, Harness: harnessName}, NeedsFirstPrompt: true}
		if a.sessions[folder] == nil {
			a.sessions[folder] = make(map[sessionKey]*harness.Session)
		}
		a.sessions[folder][sessionKey{harness: harnessName, id: id}] = s
	}
	a.mu.Unlock()
	if s == nil {
		apiWriteError(w, http.StatusNotFound, "session not found")
		return
	}
	var model *harness.Model
	if req.Model != "" {
		a.mu.Lock()
		cached, ok := a.models[s.Harness][req.Model]
		a.mu.Unlock()
		if !ok {
			apiWriteError(w, http.StatusBadRequest, "unknown model")
			return
		}
		model = &cached
		if req.Reasoning != "" {
			valid := false
			for _, level := range model.Reasoning {
				valid = valid || level == req.Reasoning
			}
			if !valid {
				apiWriteError(w, http.StatusBadRequest, "invalid reasoning level")
				return
			}
		}
	}

	// update the QueuedMsg + other stuff
	s.Mu.Lock()
	if s.IsDeleted || s.IsDeleting || s.IsStopping {
		s.Mu.Unlock()
		apiWriteError(w, http.StatusConflict, "session is being deleted or stopped")
		return
	}
	if s.QueuedMsg != "" && strings.TrimSpace(req.Prompt) != "" {
		s.QueuedMsg += "\n"
	}
	s.QueuedMsg += strings.TrimSpace(req.Prompt)
	s.QueuedImages = append(s.QueuedImages, req.Images...)
	s.PromptModel, s.PromptReasoning = model, req.Reasoning
	queued, queuedImages := s.QueuedMsg, append([]string{}, s.QueuedImages...)
	s.Mu.Unlock()
	a.setLastOpened(folder, s.ID)
	apiWriteJSON(w, http.StatusAccepted, map[string]any{"sessid": s.ID, "harness": s.Harness, "queued": queued, "queued_images": queuedImages, "busy": s.GetIsBusy()})
}

func (a *app) loadModels() {
	for name, adapter := range a.adapters {
		models, err := adapter.GetModels()
		if err != nil {
			log.Printf("adapter.models(%s) failed: %v", name, err)
			a.mu.Lock()
			if a.models != nil {
				delete(a.models, name)
			}
			a.mu.Unlock()
			continue
		}
		a.mu.Lock()
		if a.models == nil {
			a.models = make(map[string]map[string]harness.Model)
		}
		a.models[name] = models
		a.mu.Unlock()
	}
}

func (a *app) handleModels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		apiMethodNotAllowed(w)
		return
	}
	if len(a.models) == 0 {
		a.loadModels()
	}
	result := make(map[string]map[string][]string, len(a.adapters))
	a.mu.Lock()
	for name, models := range a.models {
		entry := make(map[string][]string, len(models))
		for modelID, model := range models {
			reasoning := model.Reasoning
			if reasoning == nil {
				reasoning = []string{}
			}
			entry[modelID] = reasoning
		}
		result[name] = entry
	}
	a.mu.Unlock()
	apiWriteJSON(w, http.StatusOK, result)
}
