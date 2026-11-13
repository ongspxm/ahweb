# devlog (your interface with dev)
- dev will give u tasks details thru the devlog. read just the portions relevant to ur task
- NEVER edit the devlog file without an explicit OK
- CONSISTENCY: clarify when details in the devlog doesnt match what you are asked to do

# general code style
- always keep code compact and ponytail (prefer reuse and inline by default)
- dun write stupid test. only test real logic relevant to the project
- we want to keep the global scope clean. if fns are only used in one place, nest them inside the bigger fns or inline them.
    - nested fns should start with underscore, eg `_card(...)`
- for long files, there should be `big sections` and fns should have appropriate prefixes
    - for eg, in app_harness.js, fn that draws stuff, are named renderXXX(), under `=== rending`
    - and generally, fns that make api calls are named apiXXX(), under the `=== api calls` header

# css code style
- the bulk of the css is kept in index.html
- group the declarations under appropriate headers, sort them based on "logical groupings"
- minimize declaration, group appropriate css styling under one single line

# structure and naming convention
- for adapters, always use readwrite CLI or RPC connection. never use readonly (if possible)
- if given a choice between session specific connection or daemon, always pick daemon (reduce load on system)
- configuration: all params and settings should sit in config.json (NO other mechanism, like cli flags)
- api endpoint structs should be inlined in the handleXXX fns itself.
- when it comes to `Adapter.fn(*Session)`, we generally don't want to call them willy nilly, but keep it all within the poll worker

# ui libs
- use icons wherever its appropriate (to replace words): lucide
- minimize styling, using pico.css styling as much as possible
- use these colors as much as possible: https://picocss.com/docs/css-variables#all-css-variables
- all lib files (css/js) should be loaded directly from web/lib instead of a remote address (like unpkg)

# documentation for api calls
- whenever changes are made to the output and query params of the API calls, make sure README is updated.
- same for harness.Adapter interface.
- keep description terse, only impt details on how to use the api

# main_xxx.go files convention
- each "group of features" should have their own main_xxx.go file, with the corresponding main_xxx_test.go file
- main.go is strictly for plumping for the api only. creating connection, the paths, etc etc
- any test written should not be testing the "small item", we should be testing in a "more integration test" style, so that the test touches more stuff
    - eg: when testing the handleXXX fns we should always aim to test at api request and response level, instead of testing the handleXXX() directly

# ui hotkeys, action dialog
- always group and order the actions according to some logical grouping (even for those without hotkeys)
- the ordering should be on the semantics of the action, not the keyboard shortcuts.
- IMPT: shift doesnt really work, the way we capture the keys (eg we cant do leader + ? cos ? is shift+/)
