# Go Architecture Rules

## GOCODE-001: all config must live in config.json

Trigger: store all runtime parameters in ~/.ahweb/config.json
we shouldnt be introducing settings overrides in cli / env vars
all config should be consistently controlled thru the jsonfile

Exceptions: only --port, nothing else. (so that we can enable multiple instance)

Severity: P1 if configuration sources conflict or produce inconsistent behavior.

## GOCODE-002: use main_xxx to group different features

Trigger: Put implementation in main_xxx.go and coverage in main_xxx_test.go.
this keeps the main ahweb runner clean
and helper fns are kept within the specific main_xxx files

Severity: P2 if feature implementation or tests become difficult to locate.

## GOCODE-003: main.go should only do plumbing and routing

Trigger: main.go should only do setup and connection routing
all other logic should live in main_xxx files

Severity: P2 if main.go starts including business logic

## GOCODE-004: main_xxx_test should be integration-style

Trigger: we want to have integration-style test on the main api features
this means testing handlers at HTTP request/response level
a typical test will construct a request, call the endpoint, and assert the response and relevant state.

Severity: P1 if tests pass without exercising real API behavior.
