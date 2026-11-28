# Endpoint Contract Rules

## ENDPOINT-001: inline endpoint structs in the handleXXX() fns

Trigger: nest endpoint structs (e.g. request unmarshal types) inside "handleXXX()" fns -> keep package scope clean.
Exceptions: shared wire types allowed only when genuinely shared across multiple endpoints.
Severity: P2 if payload ownership unclear or contracts drift.

## ENDPOINT-002: document all public API change in README.md

Trigger: keep "README.md" API docs synced -> match query params and state concise usage intent, strip everything else.
Exceptions: internal failure modes and invariants belong in review rules or code comments.
Severity: P1 if "README.md" API docs mismatch code; P3 if docs become verbose.
