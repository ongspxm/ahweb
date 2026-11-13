package main

import (
	"errors"
	"log"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ongspxm/ahweb/harness"
)

type pollJob struct {
	session *harness.Session
	done    *sync.WaitGroup

	// xxxResult != nil means that the poll job is for delete/rewind
	// done is for completion, xxxResult is for err + job identification

	// deleteLast is tstamp of preview for us to match preview tstamp and sess tstamp
	deleteResult chan error
	deleteLast   int64

	// rewindID is the id of the msg we want to rewind up till
	rewindResult chan error
	rewindID     string
}

func (a *app) pollLoop() {
	workers := a.config.PollWorkers
	a.pollJobs = make(chan pollJob, workers)
	for i := 0; i < workers; i++ {
		go func() {
			for job := range a.pollJobs {
				a.pollSession(job)
			}
		}()
	}
	go a.cleanupLoop()
}

func (a *app) pollSession(job pollJob) {
	defer job.done.Done()
	s := job.session
	s.OpMu.Lock()
	defer s.OpMu.Unlock()

	// perform delete
	if job.deleteResult != nil {
		var err error
		s.Mu.Lock()
		if s.IsDeleted {
			err = errors.New("session is already deleted")
		} else if s.IsDeleting {
			err = errors.New("session is already being deleted")
		} else if s.IsStopping {
			err = errors.New("session is stopping")
		} else if s.GetIsBusyLocked() {
			err = errors.New("cannot delete a busy session")
		} else if s.LastTimestampMs != job.deleteLast {
			err = errors.New("session changed since preview")
		} else {
			s.IsDeleting = true
		}
		s.Mu.Unlock()

		if err == nil {
			if adapter := a.adapters[s.Harness]; adapter == nil {
				err = errors.New("session adapter is unavailable")
			} else {
				err = adapter.Delete(s)
			}
			s.Mu.Lock()
			s.IsDeleting = false
			if err == nil {
				s.IsDeleted = true
			}
			s.Mu.Unlock()
		}
		job.deleteResult <- err
		return
	}

	// perform rewind (follow up with full history poll in same poll run)
	if job.rewindResult != nil {
		s.Mu.Lock()
		if !s.GetCanRewindLocked() {
			s.IsRewinding = false
			s.Mu.Unlock()
			job.rewindResult <- errors.New("cannot rewind a busy / deleted session")
			return
		}
		s.Mu.Unlock()
		adp := a.adapters[s.Harness]
		var err error
		if adp == nil {
			err = errors.New("session adapter is unavailable")
		} else if err = adp.Connect(s, false); err == nil {
			err = adp.Rewind(s, job.rewindID)
		}

		if err != nil {
			s.Mu.Lock()
			s.IsRewinding = false
			s.Mu.Unlock()
			job.rewindResult <- err
			return
		}
		s.Mu.Lock()
		s.DoFullHistory = true
		s.EmptyPollCnt = 0
		s.Mu.Unlock()
	}

	// do poll if still active
	a.mu.Lock()
	pollResetThreshold := a.config.NumEmptyPollBeforeCLIReset
	a.mu.Unlock()
	s.Mu.Lock()
	prevCnt := len(s.Events)
	doFullHistory := s.DoFullHistory || s.EmptyPollCnt > pollResetThreshold
	s.DoFullHistory = doFullHistory
	s.Mu.Unlock()

	// connect to make sure session is alive
	adp := a.adapters[s.Harness]
	if err := adp.Connect(s, !s.GetIsBusy() && doFullHistory); err != nil {
		log.Printf("adapter.connect(%s, %s) failed: %v", s.Harness, s.ID, err)
		s.Mu.Lock()
		s.Events = append(s.Events, harness.Event{Type: harness.EventError, Content: err.Error(), TimestampMs: time.Now().UnixMilli()})
		if job.rewindResult != nil {
			s.IsRewinding = false
		}
		s.Mu.Unlock()
		_ = adp.CleanUp(s)
		if job.rewindResult != nil {
			job.rewindResult <- err
		}
		return
	}

	// Send a new session's first prompt before polling for history.
	s.Mu.Lock()
	firstPrompt := s.NeedsFirstPrompt
	firstQueued := s.QueuedMsg != ""
	s.Mu.Unlock()
	if firstPrompt {
		if !firstQueued {
			return
		}
		a.sendQueued(s, adp)
		s.Mu.Lock()
		pending := s.NeedsFirstPrompt
		s.Mu.Unlock()
		if pending {
			return
		}
	}
	pollErr := adp.Poll(s)
	if pollErr != nil {
		log.Printf("adapter.poll(%s, %s) failed: %v", s.Harness, s.ID, pollErr)
		s.Mu.Lock()
		s.Events = append(s.Events, harness.Event{Type: harness.EventError, Content: pollErr.Error(), TimestampMs: time.Now().UnixMilli()})
		s.Mu.Unlock()
		_ = adp.CleanUp(s)
	}

	// clean up all the sess status flag
	s.Mu.Lock()
	if len(s.Events) == prevCnt {
		s.EmptyPollCnt++
	} else {
		s.EmptyPollCnt = 0
	}
	if doFullHistory {
		s.DoFullHistory = false
		s.EmptyPollCnt = 0
		if s.DoneFullHistory != nil {
			close(s.DoneFullHistory)
			s.DoneFullHistory = nil
		}
	}
	s.BusyMsg = func() bool {
		if len(s.Events) == 0 {
			// Empty sessions must stay idle so their first prompt can be sent.
			return false
		}
		if s.LastTimestampMs == 0 {
			return true
		}
		age := time.Since(time.UnixMilli(s.LastTimestampMs))
		if age > time.Minute {
			return false
		}
		for i := len(s.Events) - 1; i >= 0; i-- {
			if s.Events[i].TimestampMs > 0 {
				return s.Events[i].Type == harness.EventUser || age <= 3*time.Second
			}
		}
		return true
	}()
	if job.rewindResult != nil {
		s.IsRewinding = false
		s.BusyMsg = false
	}
	busy, queued := s.BusyMsg || s.BusyCli, s.QueuedMsg != ""
	s.Mu.Unlock()
	if job.rewindResult != nil {
		job.rewindResult <- pollErr
	}
	if queued && !busy {
		a.sendQueued(s, adp)
	}
}

func (a *app) sendQueued(s *harness.Session, adp harness.Adapter) {
	s.Mu.Lock()
	msg, imgs := s.QueuedMsg, s.QueuedImages
	// Keep the session busy while the prompt is being sent.
	s.QueuedMsg, s.QueuedImages = "", nil
	s.SendingMsg, s.SendingImages = msg, imgs
	s.BusyCli = true
	s.Mu.Unlock()

	if err := adp.SendPrompt(s); err != nil {
		log.Printf("adapter.send_prompt(%s, %s) failed: %v", s.Harness, s.ID, err)
		s.Mu.Lock()
		if s.QueuedMsg != "" && msg != "" {
			msg += "\n"
		}
		s.QueuedMsg = msg + s.QueuedMsg
		if imgs != nil {
			s.QueuedImages = append(imgs, s.QueuedImages...)
		}
		s.SendingMsg, s.SendingImages = "", nil
		s.BusyCli = false
		s.Mu.Unlock()
		return
	}
	s.Mu.Lock()
	s.SendingMsg, s.SendingImages = "", nil
	s.NeedsFirstPrompt = false
	s.Mu.Unlock()
}

func (a *app) cleanupLoop() {
	prevPollSummary, prevFolderCount, prevRootCount := "", -1, -1
	cleaned := make(map[*harness.Session]bool)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for range ticker.C {
		a.mu.Lock()
		cleanupAfter := time.Duration(a.config.SessionCleanupAfterMinutes) * time.Minute
		folderCleanupAfter := time.Duration(a.config.FolderCleanupAfterMinutes) * time.Minute
		roots := strings.FieldsFunc(a.config.GitRootRealDirs, func(r rune) bool { return r == '\n' || r == '\r' })
		lastFolderCleanup := a.lastFolderCleanup
		if time.Since(lastFolderCleanup) >= folderCleanupAfter {
			a.lastFolderCleanup = time.Now()
		}
		a.mu.Unlock()
		if time.Since(lastFolderCleanup) >= folderCleanupAfter {
			activeFolders := make(map[string]bool)
			refreshOK := true
			for _, root := range roots {
				dirs, err := gitDirs(root, true)
				if err != nil {
					log.Printf("folder.cache refresh failed: root=%s error=%v", root, err)
					refreshOK = false
					break
				}
				for _, dir := range dirs {
					activeFolders[dir.Dirname] = true
				}
			}
			if refreshOK {
				a.mu.Lock()

				// figure out which folder to remove
				folders := make(map[string]map[sessionKey]*harness.Session)
				for folder, byID := range a.sessions {
					if !activeFolders[folder] {
						cached := make(map[sessionKey]*harness.Session, len(byID))
						for key, s := range byID {
							cached[key] = s
						}
						folders[folder] = cached
					}
				}
				a.mu.Unlock()
				a.cliMu.Lock()
				for folder := range a.cliSessions {
					if !activeFolders[folder] {
						if _, exists := folders[folder]; !exists {
							folders[folder] = nil
						}
					}
				}
				a.cliMu.Unlock()

				// actually do the cleanup
				for folder, byID := range folders {
					var succeeded []sessionKey
					for key, s := range byID {
						adapter := a.adapters[s.Harness]
						if adapter == nil {
							log.Printf("adapter.cleanup(%s, %s) failed: adapter missing", s.Harness, s.ID)
							continue
						}
						s.OpMu.Lock()
						err := adapter.CleanUp(s)
						s.OpMu.Unlock()
						if err != nil {
							log.Printf("adapter.cleanup(%s, %s) failed: %v", s.Harness, s.ID, err)
							continue
						}
						succeeded = append(succeeded, key)
						delete(cleaned, s)
					}
					a.cleanupCLISession(folder)

					// update all the cache
					a.mu.Lock()
					for _, key := range succeeded {
						if a.sessions[folder][key] == byID[key] {
							delete(a.sessions[folder], key)
						}
					}
					if len(a.sessions[folder]) == 0 {
						delete(a.sessions, folder)
						a.lastOpened.Lock()
						delete(a.lastOpened.times, folder)
						a.lastOpened.Unlock()
					}
					a.mu.Unlock()
				}

				// logging only when needed
				if len(activeFolders) != prevFolderCount || len(roots) != prevRootCount {
					prevFolderCount, prevRootCount = len(activeFolders), len(roots)
					log.Printf("folder.cache active_folders=%d root_folders=%d", prevFolderCount, prevRootCount)
				}
			}
		}

		a.mu.Lock()
		var sessions, cleanupSessions []*harness.Session
		polledByHarness := make(map[string]int)
		for _, byID := range a.sessions {
			for _, s := range byID {
				s.Mu.Lock()
				doFullHistory, hasEvents := s.DoFullHistory, len(s.Events) != 0
				lastTimestamp := s.LastTimestampMs
				s.Mu.Unlock()
				poll := doFullHistory || s.GetIsBusy() || !hasEvents
				if poll {
					sessions = append(sessions, s)
					polledByHarness[s.Harness]++
					cleaned[s] = false
				} else if !cleaned[s] && lastTimestamp > 0 && time.Since(time.UnixMilli(lastTimestamp)) >= cleanupAfter {
					cleanupSessions = append(cleanupSessions, s)
				}
			}
		}
		a.mu.Unlock()
		parts := make([]string, 0, len(polledByHarness))
		for h, c := range polledByHarness {
			parts = append(parts, h+":"+strconv.Itoa(c))
		}
		sort.Strings(parts)
		summary := strings.Join(parts, ", ")
		if summary == "" {
			summary = "0"
		}
		if summary != prevPollSummary {
			prevPollSummary = summary
			line := "poll.session " + summary
			a.mu.Lock()
			a.pollStatus = line
			a.mu.Unlock()
			log.Printf("%s", line)
		}

		var done sync.WaitGroup
		for _, s := range sessions {
			done.Add(1)
			a.pollJobs <- pollJob{session: s, done: &done}
		}
		done.Wait()

		// Clean up in a second loop because it blocks; cleaned prevents repeated calls.
		for _, s := range cleanupSessions {
			s.OpMu.Lock()
			err := a.adapters[s.Harness].CleanUp(s)
			s.OpMu.Unlock()
			if err != nil {
				log.Printf("adapter.cleanup(%s, %s) failed: %v", s.Harness, s.ID, err)
				continue
			}
			cleaned[s] = true
		}
	}
}
