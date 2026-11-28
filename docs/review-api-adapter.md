# Adapter Implementation Rules

## ADAPTER-001: code pattern to map harnessID to sessionID

Trigger: harnessID differs from sessionID -> standardize mapping for chat deduplication.
Invariant: each adapter defines "harnessIDToSessID[realFolder][harnessID]" and "sessIDToHarnessID[realFolder][sessID]":
1. On session create -> read harnessID returned for ahweb sessID, save in both maps.
2. On list chats -> unseen harnessID creates new SessionLite; seen harnessID reuses original sessID.
3. On adapter action needing native ID -> read "sessIDToHarnessID".
Verify: check maps save on create, dedupe on list, resolve on action; repeated harnessID yields same sessID.
Exceptions: if harnessID equals local sessID, both maps may map value to itself.
Severity: P2 if duplicate chat appears or action hits wrong chat.

## ADAPTER-002: implement using daemon shared across sessions, as opposed to session specific connections

Trigger: prefer shared daemon connection across sessions over per-session connections -> cuts process and transport overhead.
Exceptions: per-session connections allowed when harness lacks daemon.
Severity: P2 if connection setup/teardown materially slows performance.

## ADAPTER-003: always call Adapter.fn(Session) within the main poll loop

Trigger: run all session Adapter calls (e.g. SendPrompt, CleanUp) inside main poll loop -> one adapter call per session at a time, avoids concurrency complexity.
Verify: search direct adapter calls -> occur only in poll worker. No exceptions.
Severity: P1 if concurrent or inconsistent session state possible.

## ADAPTER-004: no test files in adapter packages

Trigger: keep adapter logic clean and aligned with Adapter interface -> zero per-adapter unit tests.
Verify: glob "adapters/**/*_test.go" -> zero matches.
Severity: P2 if tests live under "adapters/".

## ADAPTER-005: map all supported EventTypes and Event attributes from real harness output

Trigger: adapter parses harness stdout or transcript into events.
Invariant: assistant/thinking model and reasoning match source turn; "Event.Model" matches "GetModels()" IDs without adapter-name prefixes.
Verify: inspect real harness stdout or transcript sample -> payload event types/fields map or show 0/? in "README.md":
1. Live and full-history polling map all supported "EventType" and event fields.
2. Populate model, reasoning, and "RewindID" from source turn.
3. Assistant/thinking "Event.Model" matches "GetModels()" IDs; compose provider/model only when source gives both.
4. Keep missing turn metadata empty; never backfill old turns from current settings.
5. Mark unsupported/partial fields 0 or ? in "README.md" feature table -> zero silent drops.
6. Map source errors -> "EventError" in live and full-history paths; error-only messages survive filters with source message/status intact.
Exceptions: CLI/protocol omits field or event type (marked 0/? in "README.md"; leave metadata empty).
Severity: P2 if thinking, images, model/reasoning, or errors drop silently, or "Event.Model" mismatches "GetModels()".

## ADAPTER-006: always keep Adapter interface document updated in README.md

Trigger: keep "README.md" Adapter docs synced with code -> cover new fns, return values, or changed behavior.
Severity: P1 if adapter implementation mismatches "README.md".

## ADAPTER-007: stop owned CLI process trees with processgroup

Trigger: adapter starts owned CLI via "cmd.Start()" (streaming, long-lived, or timed probe) that may spawn children -> use "internal/processgroup": "processgroup.Start(cmd)" isolates pgid; "processgroup.Stop(cmd, done, grace)" or "processgroup.Kill(cmd)" stops on stop, cleanup, timeout, or startup fail; wait for process reap and output workers ("done" / "cmd.Wait()") before dropping session state.
Verify: trace every "exec.Command"/"Start" + stop, cleanup, timeout, and error path -> uses "processgroup.Start" and "processgroup.Stop"/"Kill", reaps process, finishes output workers before clearing state.
Exceptions: shared/detached daemons ("nohup", "daemon start", "service start"), sync one-shot "Output()"/"CombinedOutput()" commands without children, or RPC cancel that reliably stops child work.
Severity: P1 if stopped work keeps running; P2 if child processes only leak.
