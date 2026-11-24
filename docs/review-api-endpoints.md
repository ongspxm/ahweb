# Endpoint Contract Rules

## ENDPOINT-001: inline endpoint structs in the handleXXX() fns

Trigger: we want to avoid populating the package scope.
therefore we keep any endpoint related structs nested in the handleXXX() fns.

Verify: common examples include structs used for unmarshalling the requets

Exceptions: Shared wire types may be used only when genuinely shared by multiple endpoints.

Severity: P2 if payload ownership becomes unclear or contracts drift.

## ENDPOINT-002: document all public API change in README.md

Trigger: we want to keep the api endpoint documentation consistent.
make sure the query param matches and provide short desc of the intent.
only keep usage-oriented details, removed everything else.

Exception: internal fialture modes and invaraints below in review rules / code comments.

Severity: P1 if API document in README.md doesnt match code.
P3 if documentation becomes verbose
