# Go Architecture Rules

## GOCODE-001: all config must live in config.json

Trigger: store all runtime parameters in "~/.ahweb/config.json" -> zero CLI or env var overrides.
Exceptions: "--port" only (enables multiple instances).
Severity: P1 if config sources conflict or diverge.

## GOCODE-002: use main_xxx to group different features

Trigger: put feature code in "main_xxx.go" and tests in "main_xxx_test.go" -> keeps runner clean and helpers local.
Severity: P2 if features or tests hard to locate.

## GOCODE-003: main.go should only do plumbing and routing

Trigger: "main.go" handles setup and routing only -> all business logic lives in "main_xxx.go".
Severity: P2 if "main.go" accumulates business logic.

## GOCODE-004: main_xxx_test should be integration-style

Trigger: write integration-style tests in "main_xxx_test.go" -> build request, call endpoint, assert HTTP response and real state.
Invariant: tests assert real HTTP response and state, not mocked internals.
Verify: test handlers via HTTP request/response; for live adapters or persisted transcripts, start ahweb on unused side port -> test "POST /api/prompt" live polling and "GET /api/events?start_from=-1" full-history polling -> check all supported event types/fields and compare models with "GET /api/models" -> stop server and delete only test sessions.
Exceptions: live side-port check only when handler tests cannot reach adapter or persisted-data path.
Severity: P1 if tests pass without exercising real API behavior.
