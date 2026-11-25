# Adapter Implementation Rules

## ADAPTER-001: code pattern to map harnessID to sessionID

Trigger: We want to have a standardized pattern for
when harnessID differs from the sessionID
this to for session deduplication when listing chats

Invariant: In each adapter, define both:
`harnessIDToSessID[realFolder][harnessID]` and
`sessIDToHarnessID[realFolder][sessID]`
1. when we create new session, read the harnessID returned by the harness that maps to the ahweb created sessID
and save it in both maps
2. when listing chats, if we see a new harnessID we havent seen before
this means we havent had a sess for it yet so we create a new sessionLite for it
if we seen the harnessId already, means that its duplicated, so we use the original sessID
3. when an adapter action needs the harness/native ID, read `sessIDToHarnessID`

Verify: Double chk that adapters save harnessIDToSessID when they create sessions
and use it when listing sessions. Double chk that adapters save
sessIDToHarnessID and use it for actions that need the native harness ID.
Make sure repeated harnessID returns the same sessID.

Exceptions: if a harness ID is also the local sessID, the two maps may map
directly to the same value.

Severity: P2 if a duplicate chat appears or an action affects the wrong chat.

## ADAPTER-002: implement using daemon shared across sessions, as opposed to session specific connetions

Trigger: we prefer a daemon/shared connection over session-specific connections.
this reduces the resources used per adapter, be it process or transport.

Exceptions: Session-specific connections are permitted when the harness provides no daemon.

Severity: P2 if excessive connection setup or teardown materially reduces performance.

## ADAPTER-003: always call Adapter.fn(Session) within the main poll loop

Trigger: the adapter is designed such that all actions happens within the poll loop (eg SendPrompt, CleanUp)
this make it such that there is only ever one adapter fn call acting on a single session at one time
therefore adapter fns for session can avoid dealing with concurrently. this make code simpler.

Verify: Search for direct adapter calls and confirm they occur only in the poll worker. no exeptions.

Severity: P1 if concurrent or inconsistent session state may result.

## ADAPTER-004: no test files in adapter packages

Trigger: Keep adapter logic clean, align with the Adapter interface. No need per-adapter unit tests.
Glob `adapters/**/*_test.go` and confirm zero matches.

Severity: P2 if tests live under `adapters/`.

## ENDPOINT-003: always keep Adapter interface document updated in README.md

Trigger: keep the README updated of any Adpater changes
this can be new fns, new return values
or different behavior expectation.

Severity: P1 if specific adapter implementation dun match the readme documentation
