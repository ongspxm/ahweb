# commands
- run: `go run .` (default port `8321`; `--port 9090` to override)
- build: `go build ./...`
- test: `go test ./...`
- hooks: `scripts/setup-githooks.sh`

# boundaries
- NEVER edit `docs/devlog.md` without an explicit OK
- no test files under `adapters/` (`adapters/**/*_test.go`)
- update `README.md` when public HTTP endpoints or `harness.Adapter` interface change

# devlog (your interface with dev)
- dev will give u tasks details thru the devlog. read just the portions relevant to ur task
- CONSISTENCY: clarify when details in the devlog doesnt match what you are asked to do

# code
review rules are included as docs/review-xxx.md
they should roughly follow the trigger, invariant, verify, exception, severity structure

when reviewing, list all the review rules mdfiles that you read in full.
(so we know which review rules you used)
list all violation in this format: violation area, reason, wut's next

## general code style
- always keep code compact and ponytail (prefer reuse and inline by default)
- dun write stupid test. only test real logic relevant to the project
- we want to keep the global scope clean. if fns are only used in one place, nest them inside the bigger fns or inline them.
    - nested fns should start with underscore, eg `_card(...)`
- for long files, there should be `big sections` and fns should have appropriate prefixes
    - for eg, in app_harness.js, fn that draws stuff, are named renderXXX(), under `=== rendering`
    - and generally, fns that make api calls are named apiXXX(), under the `=== api calls` header
    - in adapters, group fns under `=== adapter admin fns`, `=== session admin fns`, `=== session fns`, and `=== util fns`

## webui
review webui changes against all the docs/review-webui-xxx.md

## gocode
review api changes against all the docs/review-api-xxx.md

## discussing code structure using mermaid diagrams
when discussing Adapter's code, its possible to use seq diagram
each of the Adapter fns will be a par scope, named like "Connect()" / "CleanUp()"

(we want to know how things are being updated)
each of the "things that store data", eg conn.Events session.Events etc etc
should exist as a participant, the arrows will show "how the data storage is altered"
also add a user to identify triggering Adapter fns, like "Connect()"
