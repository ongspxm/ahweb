# devlog
entries should roughly be along the lines of: explain why the change
describe current situation, what should be the new change
roughly wut the shape of the change look like in each of api/webui
okay to split multiline, one point one line. (look at past entry for eg)

states: TODO for new, DONE for completed, PARK for in the future. REJECT for dun fix.
ORDER: older entries at the bottom

migration means we use a clean devlog.md will all these frontmatter copied over
and save the old one as YYYYMMDD-devlog.md where YYYYMMDD is the most recent date of that file.
NOTE: when migrating, we move over the TODO and PARK notes (MOVE not COPY)

potential backlog:
- fork

## 2026-11-11 DONE. render webui mermaid diagram
we want to render mermaid diagrams on webui
for all code rendered as mermaid text from "assistant msg"
add a btn that swaps between "text" and "diagram"

this should be a webui side only change.
use mermaid-js (official package is lightweight enough for use)
it should work alongside the existing markdown rendering for "assistant msg"

## 2026-11-11 DONE. save completed tours on server
we want to be able to skip the tours for this server if we did it already
even if on another device.

fix: completed driverjs tours now saved to /config

## 2026-11-11 DONE. refresh models from workspace button
webui the refresh btn should also call /models
to refresh the nodel and harness dict with the latest copy.

## 2026-11-11 DONE. mobile project/worktree breadcrumb
on mobile. add a breadcrumb bar above the tabs.
it should show "(proj dir basename) > (worktree branch)"
display "(empty)" for anything not selected

when i click on the project dir basename, it should open sess panel, projects part
similar for worktree branch name

## 2026-11-11 DONE. webui worktree status should skip child pending
we can have many child sessions, we dun always read it
we dun wan the worktree to keep showing pending, we are ok for subsessions to be unread.
but any session that's busy should mark worktree busy

## 2026-11-11 DONE. opencode session wrong token usage
webui. opencode session showed 0% token usage in webui
dig: context tokens needs to be updated using last completed assistant message,
refactor: model limits also loaded once at server init via loadModels()

## 2026-11-10 DONE. populate opencode tool name
opencode adapter shows the toolname as generic "tool" in webui
dig: should have read "name" from the sse_event stream

## 2026-11-10 DONE. update PATH during ahweb install
we want to have the "closest experience" to terminal use for ahweb

update ahweb install so that in both linux and mac
it will update the PATH for launchd / systemd

install of running a which on the executables, we will instead
just use the PATH of the active running session
and have some sort of log line like "copying over PATH"

## 2026-11-09 DONE. some ui fixes on file tab
the file tab, the bottom part
on the left of the title, add a file icon
this file icon should link to the raw file
(actual link so that i can right click open in new tab)

also: is there a way for the /file so that
when i download it, it gives the original filename?

## 2026-11-09 DONE. keep CLI sockets alive in background
webui keeps each worktree's CLI WebSocket open when switching tabs or worktrees.
Output continues into that worktree's xterm; reconnect after a dropped socket.
Refresh clears the xterm and reconnects from the server buffer; shell and buffer stay alive.

## 2026-11-09 DONE. resizable file and git tab
webui. i want to file and git tab
the view to be adjustable, can drag up and down to resize

currently the panels have similar features.
okay to use the same mechanism

## 2026-11-09 TODO. rewind feature
most harness should support the rewind feature
(aka jump history back to a particular msg)
to simplify implementation, on ahweb we only rewind back to "any past user msg"

webui side. on each of the user msgs, will now have a small btn at the top
rewind is per message; hide the button when an event has no rewindID.
when clicked it will send a /rewind api call
the api call should return the "text prompt" and "full event history"
if busy or rewind fails, return an API error and show it with setError()

the text prompt will be used to populate the prompt textbox
(similar to the stop btn, also restore imgs)
and the event will be full rendered too
(kinda similar to refresh btn)

api side, to enable this feature, now Event can have a "rewindID"
and its up to adapter to populate as it deems fit.
Adapter.Rewind(Session, rewindID) will only return when adpater is confident that rewind is done

/rewind will basically call Rewind(), wait for it to be done
then call something like /events?start_from=-1 ish
best if actually called directly (so dun have duplicated code)
but let's see if we can pipe something simple tgt, else okay to duplicate code.

research notes (installed versions; online docs):
Pi 1.4.0: UI tree navigation by entry ID; no documented RPC rewind. https://github.com/earendil-works/pi/blob/main/packages/coding-agent/docs/rpc-commands.md
Prime 0.10.0: daemon navigate_tree by entry ID; best fit. https://github.com/PrimeIntellect-ai/prime-agent/blob/main/crates/pa-daemon/src/branch_navigation.rs
Claude Code 2.1.319: UI /rewind only; no documented headless rewind API. https://code.claude.com/docs/en/checkpointing
Codex 0.166.0: rollback uses turn count, so mapping a user message is unreliable. https://github.com/openai/codex/tree/main/codex-rs/app-server
OpenCode 2.2.0: API revert by message ID; returns 409 when busy; best fit. https://opencode.ai/docs/api
Cursor: UI /rewind, but no documented API or target message ID. https://prod.cursor.com/docs/cli/reference/slash-commands

## 2026-11-09 DONE. provide more status on running server
rethinking the events update
currently the updates only show a one-liner on webui.
was thinking of maybe showing more?

/status seems useful.
like wut's the current version running, how to update, etc etc
which harness is ok? which harness is not?
and how many root folders and how many folders
and each folder how many sessions etc

running text logs? maybe dunnid?
if we have a featureful enough ahweb binary, we might not need?
we can just tell people to "ahweb logs" instead?

on the ui, it should be another "status" tab
(prob dunnid hotkey its fine)
but include it in the action, alongside the "tab btns"

## 2026-11-08 DONE. webui cli tab status
i wanna know the status of the cli tab.
when the ws is connected and running, vs its disconnected
if the ws for the cli is active, then add a solid dot state (the pending message status dot) on "cli tab"

if the ws is disconnected, then the dot should dissapear.

## 2026-11-08 PARK. webui seems abit slow
EDIT: prob not work doing for now, it complicates state handling on server side
only really do when we actually measure that its slow.

i think its becos it keeps calling the same /events all the time.
(which most times return the same state ish? and when busy=True, it will keep calling... much overhead?)
what can i do instead to make it faster?

websocket? but the mechanism for that seems to have a lot of overhead.

let's use long polling instead? /events will keep returning items, broken by newline.
every line is a jsonl, whatever that's returned now.
it will only close the conn when finally busy=false

else it will only update when there is any new event or when session state changes.
api side, maybe: on each session add a notification channel
it will keep pushing out new events/updates and be closed on busy=false
/events can loop on that channel instead.

webui update instead of having a loop that keep polling
just keep one connection then loop to parse the returns instead.

## 2026-11-08 DONE. delete old sessions
we want to delete old sessions to keep the api cache light
Adapters expose CanDelete() and Delete(*Session).
Delete() must be run thru the poll worker.

split the api into 2 steps, first step gets the preview using confirm=0
Returns {"planid":"...","data":{"foldername":{"delete":N}}}.
expires in 2mins.

after getting the preview, we can then execute the plan using confirm=1
recheck session before actually performing deletion

Web UI: Delete old sessions button, will open up dialog with days= input (default 8),
preview table, and confirmation with a permanent-deletion warning.

## 2026-11-07 DONE. fix CLI predictive echo
the CLI uses a local PredictiveEchoAddon bundled in web/lib/xterm/xterm.js
to show printable input before the server echoes it.
app_cli.js calls the addon directly and reconciles predictions against terminal output.
allowProposedApi is enabled so the addon can use terminal.unicode for cell widths.

## 2026-11-07 DONE. auto reconn when webui cli drop.
now cli tab keeps showing "cli disconnected" randomly
the reason is because websocket is being dropped

let's implement some sort of reconnect and redraw for cli tab
it should retry with exponential backoff.
note: use offset= so that we can replay from the correct chars.

## 2026-11-07 DONE. use server side for svg instead
we want the same svg for all webui connected to the same api.
use server identity, not the webui hostname and port.

add a /host, it should return normalized hostname+user+port+uname
lowercase and underscores only. same logic for rendering svg

## 2026-11-07 DONE. handle when harness is not ready
we need some sort of way to know if the harness is ready for use

Add `Ready() error` to `harness.Adapter`; nil means ready.
Startup logs errors and leaves unavailable adapters disabled until restart.

this means we have to look thru the adapter and decide
the best readiness check for each adapter.

## 2026-11-07 PARK. integration test for adapters
kinda like "full on" simulated integration test
with the gocode as blackbox, and match api calls and resp

but that means we need to replay stuff from adapter to harness
means we need to build some "hub" to capture all the rpc/html/cmd
but the hub nid to be designed to capture all required states

might be a big piece of work.
not that valuable if we are not gonna make that much changes
but will be good to have, if this project is arnd in the long run.

## 2026-11-07 DONE. table overview of adapter and features
look at all the adpaters.
add a markdown table to the readme
each row is one feature, each col one adapter

feature will include things like "have cost" or like "have % of context window"
look at all the adapter. come up with a list of features.
they should be marked with Y/N/?

Y means fully supported, N means no support, ? means partial?
make sure its consistent, if there is a N/?
then check that there is a note for that adapter in the README section

## 2026-11-06 DONE. consistent texting behavior with other ui
webui. prompt textbox, current shift+enter submits then enter add new lines.
other UI have this swapped. let's swap this behavior as well.
but on mobile, "enter" doesnt submit.

## 2026-11-06 DONE. installation and updates
install and update with `go install github.com/ongspxm/ahweb@latest`; requires Go 1.24 or newer.
no CI-built binaries or release needed.
still check for updates and announce when one is available, with the install command.

## 2026-11-06 DONE. auto-start
start ahweb at boot, without requiring login.
(ahweb install will add it to the system)
the rest of the cmds should ideally pipe to some cmd call directly (think: fork)

add `ahweb install`, `status`, `logs`, `restart`, and `uninstall` commands.
Linux: use a systemd user service with lingering.
macOS: use a launchd daemon that runs as the user; install needs admin permission.
note: maybe can go:build to handle different os

write me e2e tests, it should be integration style, running thru all the cmds
goal: if i run the test, i should be confident that the cmds run on the particular machine

## 2026-11-06 DONE. locally hosted lib files for webui
there is a lot of js files and css files being loaded
for libs and stuff. pull those down into web/libs/
then serve them from there.

we want to be standalone and not die due to random errs
like unpkg going down and stuff..

## 2026-11-06 DONE. driverjs walk through
basically get ready to launch webui. make 2 tours
first tour covers typical flow to "get first msg"
second tour should cover other webui features, wut the btns do, hotkeys etc

look at webui code and devlog and figure out what should make it into these 2 tours
i want the tour to be future proof
(aka future feature addition shouldnt easily spoil these tour)
tours now also adapt to mobile panels and skip desktop-only steps.

and also we shouldnt be adding alot of code to "complicate our curr codebase"
keep the tour implementation as clean and separated as possible.

## 2026-11-06 DONE. add hotkey hints to the ui
show a small label at the top of the btn as hints for hotkeys
triggered with a new btn with ? icon
press esc to close hints.

for those that are leader + X just show X in the hint
for those that are full hotkeys, put them in brackets, like "(ctrl+space)"

add it to the action list, but this "toggle hints" has no hotkeys by itself

## 2026-11-06 DONE. webui fixes
keyboard shortcuts dun work on mac (ctrl+shift+4 is screenshot wft)
change the tab to use JKL; instead, we prob dun have >4 tabs.

make the refresh btn also redraw the cli
when "cli disconnect" the stale text will stay on screen
the refresh btn should clean that up
(kinda like a tmux ctrl+b r, but ui)

add btns to cli tab so that i can use on mobile
keys like esc, tab, ctrl etc etc (copy design from termux)
also add 2 btns: one to increase cli font size one to reduce.
F+ and F- should trigger a xterm resize and init

when i click on the chat, the prompt textbox should be in focus

obs: when i sometimes send user msg, the chat panel doesnt scroll up
i suspect my msg might be too long and spoils the
"close to bottom" check.

DONE: add a loop to webui: every 30s, go and pull the latest list of chats
this is so that i can see new child chats or new chats not started
by this particular session.

## 2026-11-05 DONE. git view
i want to be able see git status changes.

api side. i want to add a endpoint that return git status
/git?folder=xxx will return the output of `git status -sb`
(basically each line will be 2 char + file name)

then /git?folder=xxx&file=xxx will return the output of git diff
if that file got no diff, or any error, we just return ""

on the webui side, we will add a new tab. called git
it will look like files tab with top and bottom.
the top will have a list (colored output of /git?folder=xxx output)
each line is a file, clickable

then the bottom will show the rendered diff
of the file that's open.

## 2026-11-05 DONE. webui hotkey support
i wan to be able to support hotkeys.

There are a few kinds of control
Set these for navigation.

ctrl+enter = "new chat".click()
ctrl+space = "stop chat".click()
ctrl+shift+1 = focus on the "send prompt"
ctrl+shift+2 = "files tab".click()
ctrl+shift+3 = "cli tab".click() + "xterm cli".click()

I wan to do tiered kind of controls
Aka, these will start with a leader (like in vim)
Leader defaults to Ctrl+' (ctrl + single quote)


Leader + p = focus on the curr project btn
Leader + w = focus on the curr worktree btn
Leader + c = focus on the curr chat btn

Then there is a set that is used for the btns in the top bar
The top bar currently have 3 btns, "add project", "switch" and "theme"
They should be in ordered from left to right

Leader + 1 = first btn (I think is add project)
Leader + 2 = second btn (...)
Leader + 3 = ...

Also: add a 4 btn there with a action icon that will pop up the actions dialog

(More about the actions dialog)
The top section will show the hotkeys
Basically a list of btns with with actions aligned left and muted "next key on the left"

Eg.
"Go to Project            p"
"Go to Chat            (Ctrl + Shift + 1)"
...

Brackets are for hot keys that are "from scratch" and doesn't use the leader.

At the bottom (scrollable) add a section that allows user to change the leader.

Mechanism: when leader is pressed the actions dialog will pop up, then if we follow up with another key, the dialog will close and the action will be performed.
Eg: I press leader, the action dialog pops up, I press p and then the dialog closes and the the first project btn is selected.

Future extension: actions in the action dialog that doesn't have a hotkey

This should be a pure webui side change. Put it in a app_hotkey.
All changes should be stored in localhost, never synced to server

## 2025-11-05 DONE. mosh like prediction for cli
on slow network, the cli is laggy and hard to use
add lightweight lib so taht we have predictive echo in cli tab

## 2026-11-05 DONE. new endpoint to remove projects
the api bit:
we have /addproject endpoint. add a /delproject endpoint
it should remove the project from the list of root folders.

instead of re-writing code to do the cleanup of the folders again
set the timestmap=0 so that it will rerun again on the next cleanup loop run.

on the webui, the project btn should have some way to show delete. suggest a ui

## 2026-11-05 DONE. cleanup old folders from time to time in api
i wan to reduce cache on api server.
run a loop (configurable, default every 5mins)

when this loop run, log out the final number of "active folders"
skip log if no change.

loop should go thru the git root folders
relist the active folders. those folders that are not active anymore
we go in and Cleanup() all the chats
then remove the folder from the main_xxx cache

this should help to remove folders (which are outdated old worktrees)
note: and also clean up the cli session (for webui cli tab)

## 2026-11-05 DONE. helper hint for redrawing CLI
add a small line that says soemthing along the lines of
if cli disconnect, use F+ and F- to redraw the cli
the desktop it should appear on the left of the 2 btn: (text) (F+) (F-)
then on mobile it should appear at the top of the 2 btn rows
compact small font maybe

## 2026-11-03 DONE. CLI tab
i want cli access

API side. I want each worktree to have their own terminal. The cache will be per realfolder, and points to a CLISession
Each CLISession holds a PTY session (creak/pty), and a 256kb rolling ring buffer to store pty output.

New API endpoint /cli?real_folder=xxx
This returns a websocket connection (coder/websocket)
If no CLISession, will init one
If already have one, will pipe pty ring buffer to output

All ws inbound msgs will be piped to CLISession verbose. Output will be streamed to websocket as output verbose as well.

Add a config for default shell to run, defaults to bash.
All the cli relevant code should be in main_cli.go, new file.

Terminal side. Add a new tab called CLI, it should show a terminal pane (xterm.js) it should capture all keystrokes then pipe it over the websocket to the API. Note: we also want to handle resize and all.

Check the new dependencies listed here evaluate if they are the most modern lightweight variant of whatever they are supposed to do.

## 2026-011-02 DONE. webui add files tab
The API is already done and should be sufficient.
The files UI should be one of the tabs in webui.
The files tab should be split into 2 parts, top and bottom.

Top part display directory listing.
Dirs appear first, then files
Then click dirs to open, and reveal files inside
Nested should work.
(Note: double check the API shows hidden dotfiles as well)

Bottom is file view.
If it's text file then display with overflow-y=scroll.
If it's img/video/any format browser can embed file then html native display.
Else it binary or anything else, then say binary can't display

Text file display have a margin on the left side, to show line numbers.

## 2026-10-28 DONE. adding folders + worktress + chats split
changes on both the api and the webui

(for api: first stage just raw folder, second stage is worktrees)
in api, now i want to store a list of "gitRootRealDirs"
this list will be saved in config, newline separated list
on startup will go and populate the stuff for all these folders (main and worktrees)
when i "open a new folder", i will run cli git, the config will only save the git root tree
but will the git worktrees will be processed as well

ui side:
now the session panel is one big vertical panel
split into 3. the top (called "projects") is to list all the git root real dirs
then the middle panel is call "worktrees", this will list all the worktrees for that project
then last one is called chats, this will list the chat of the "specific worktree" (like wut we have now)

im thinking we have 3 part, each part is top title, below content
clicking on the title folds the content.
the effect i wan is: all opened parts should try to take up most of the remaining space
best if pure css (see if doable)

add a btn beside the curr webui light/dark btn
it be used to add projects to the api
is the files endpoint so that i can add dirs to the project list

on mobile. only one part can be opened at one time, the rest will be folded.

## 2026-10-29 DONE. parent and child sessions
we wan to display sub sessions.
this means api need to expose some kind of parent_sessid
but most of the render logic will be on webui

note: webui should only show max one depth
all those with no parents will be level0
then evetything else level1

on sess panel, we will use the same sorting metrics at both levels
tstamp=0 first. else, desc tstamp

## 2026-10-28 DONE. add svg favicon
we want to be able to recognize which server is running.
we can do that thru favicon from svg
make it 3char, that the host name
then the first letter upper-cased, that will be the first char
the second and third char is obtained by adding all the ord(c) of every char in hostname
then mod 99, zero padded 2 chars
forming the 3char svg favicon

## 2026-10-28 DONE. Make mobile version of webui
we want to make mobile version of webui

Current: there is a panel on the left for listing the chats, then a panel on the right for rendering the chat messages.

Non-mobile layout
Current panel for listing sessions, rename it #panel-sessions
Current panel for rendering chat, rename it #panel-chats
Add a new panel name it #panel-workspace, we will later add tabs to this for other features.

It should have non mobile default layout of left is panel-session (default 20rem), right is #panel-chat (default 30rem), the center is panel-workspace (rest of the space), but empty for now.

Panel-workspace will have a horizontally scrollable tab bar at the top
which will show all the tabs. For now, put in tabs1 and tabs2. Then create #tab-1 and #tab-2 in panel-workspace below the tab bar.

On mobile view. There is one tab bar at the top, first tab is sess, second is chat, then the rest of the tabs from panel-workspace.
Then at the bottom will display whatever tab is active.

## 2026-10-26 DONE. move last_opened to server side
I want to know which messages are new when I open a new webui session. To do this, we shift the logic for "last_opened" onto API side

Currently. Last_opened is only tracked on webui. (How it's used for computation and display won't change)
New. On main.go, we will init a new map[real folder][sessid]timestampMs called session LastOpened
If reading from map and key doesn't exist, init to serverStartTimestampMs

LastOpened stays outside Session; its map reads and writes use app.mu. Session event access keeps its existing Session.Mu protection.

Everytime /prompt is called, the current timestamp will be written to it. This last opened timestamp will be return on /list
/events will now take in a param called opened, a bool. Webui will send opened=1 if the chat is the currently opened one, else opened=0.
When opened=1, we will update the last_opened to curr timestamp.

Webui will no longer set last opened, /events will return it and the update new done in UpdateChat()

## 2026-10-20 DONE. support sending imgs to chat.
we want to be able to send imgs thru the webui.

these will happen in 4 stages, 2 for reading, 2 for writing, each major chunk, split into api and web ui.

1. api reading of imgs: /events should somehow be able to return imgs, maybe a link
    - DONE: let's build a /file and /files endpoint first
    - DONE: then we make /events return imgs as inline data URIs
2. DONE: webui render of imgs from prev chats.
3. DONE: api receiving imgs from /prompt. implemented as PromptMsg vs SendingMsg and PromptImages and SendingImages
4. DONE: webui form to send imgs

## 2023-10-24 REVIEW. ui fixes
x sess list need to update busy status
x sess list needs to stop re-ordering the session
x remove ugly loading bar for chat
x pi adpater needs to read error msgs
x coloring the activity for succ vs error

## 2023-10-23 DONE. double check connection on Connct() for prime adapter
we want to fix the case where prime session sometimes expire
convention: after Connect() is called, the connection should be live and ready to use.

## 2023-10-23 DONE. webui make thinking part of the activity block
we want a more compact view.
instead of closing the activity block when there is a thinking block
treat the thinking block as an activity so that the final chat look more compact.

## 2023-10-23 DONE. webui stop scrolling down all the time
when i reading past chats, and new msgs come in
leave it, dunnid to scroll to the bottom

## 2023-10-23 DONE. webui only load chats it needs, so that we reduce mem usage on browser.
webui should only load the chats it needs
because now full chat can be very heavy, with imgs and all.
server continue having everything loaded, ready to reply

## 2026-10-23 REJECTED. use pi in the daemon pattern.
pi doesnt native have a daemon, if we want to do that, we will have to ship some sort of
pi-daemon then connect using unix sockets, but then we will have to replicate all the "state taking care" logic
so let's leave that aside for now

## 2026-10-23 DONE. handle "display of name" for new sessions
when we start new sessions, we want the webui to display its name, instead of just the sessid

prev implementation was messy, it oni did the 140char population of the name in listSesions
now we added a explicit FirstUserMsg, and that will be used in main.go when we need the sesion name
then all the adapters can cleanly populate the fields and we do the 140chars hacky oni in one place.

## 2026-10-23 DONE. fix harness for new session is wrong err
bug: when i start new session from webui, i start in prime, but the harness displayed is pi
reason: we dun return harness as part of /prompt so have no info and it fallback to "pi"

## 2026-10-21 DONE. add prime-agent adapter
prime agent should always communicate using daemon. send to one instant to reduce load on the system.

some notes: listSession do pure file reading, just like in pi
also updated webui to support multiple harness
when daemon is down on Connect(), we will have logic to start it up (dial loop)
CleanUp() might actually kill the session (if we are the only one using it -- aka stop the prime ipynb)

## 2026-10-21 DONE. move handleFile to main_file

we want to have cleaner split of code.
esp later when main will have different features, like git and terminal
so we are moving "feature specific stuff" to different main_xxx files
most of the harness stuff will move into main_harness (later)
and main.go will be strictly used for plumping only

## 2026-10-21 DONE. update settings to return full set of settings
we want to be able to pull all settings without having to do it one-by-one in multiple queries
when key=xxx param not set for /settings, it will return the full dict.

## 2026-10-21 DONE. support /file and /files endpoint
we want this so that we can support generic loading of files from the webui, which enables img loading and sending

we just need to have a /file endpoint, one single endpoint
if its a dir, it will return json of the dir listing
else it will serve the raw file
if not found, then it will err

original design:
/file endpoint that can return raw files from the system
and a /files endpoint that do file listing.
both should work relatively to realpath folder

## 2026-10-20 DONE. webui to show "pending msg" + "busy"

we want to be able to reflect chat session status in the webui.

current: its a progress bar at the top of the session card on the sess list.
new: pulsing circle (use a pico css color) for busy. solid circle for pending messages.

logic for "pending msgs":
- when page loads, init var tstamp0 as the current time
- in webui chat, add a last_opened=tstamp0 to the chat in CACHE
- when poll rets and i happen to be on that chat, update that last_opened to the last tstamp received on the sess

logic for rendering:
- pending = last_tstamp > last_opened
- if busy, render busy, else if pending render pending, else nothing
- pulsing circle should be at the right most of the second line of the session card.

## 2026-10-20 DONE. webui suporrt reasoning levels

## 2026-10-19 DONE. app.js and gocode is in acceptable shape, we will use this as a base to build upon.

## 2026-10-12 DONE. Syncing webui and API endpoint chat cache

We want to standardize chat state syncing between webui and API.

On webui, CacheChat[realpath][sessid] = {events=[...], stats0, stats1, stats2..., queued=, busy=, tstamp=} the statsXXX refers to those token stats for the specific chat. (Hint: tstamp is last_timestamp in API)

On the webui, there should be a fn, updateChat(realpath, updatedChat) where chat is a dict with keys defined above (plus an additional sessid).

all the fields should just be replaced if the field exist in updatedChat but not in cache, except events.

For every message in updatedChat.events, it should have a int seq. We then set SpecificCacheChat.events[seq] = msg. Special case: if updatedChat.events[0].seq=0 then we will do a SpecificCacheChat.events=[] to reset first.

All the API endpoints should be this format output. Every endpoint call for specific chat should return something of this json (only included updated fields). That result should be directly passed into updateChat fn.

IMPT: no changes to the way the API internally represents data. Just the API formatting and webui.

## 2026-10-09 DONE. refactoring Model
we want to be able to "parse id back" into Model for prompt sending.

i wan to modify harness.Model
to only have Harness, ID, Reasoning, DefaultReasoning
instead of having provider and id for pi
it will have just one ID, which is "provider/id"

then GetModels will now return map[modelid]Model instead
that can be cached in main under map[harness][modelid]Model
that i can use later in handlePrompt (dunnid do now)

also: note the change from ReasoningIdx to Reasoning
we store string now. empty string means nil

## 2026-10-09 DONE. sendPrompt draft
logic for `sendPrompt(*Session)`. if s.PromptModel and s.promptReasoningIdx not set

then after that we will work on piAdapter.SendPrompt()
it will set the model and reasoning level, then push s.QueuedMsg into the rpc session as a fn call
then set busyCli=true

ui expectation: when i click the "send btn", it will first show some sort of loading (maybe replace the send button with sending?)
then when the api returns back the full queuedMsg, the btn will turn back to send, and the queuedMsg as returned by the api will be displayed.

extension of ui expectation: when i click stop (the stop btn will have its icon replaced?)
when the api return back, it will give the queuedMsg, and then we will populate the prompt text box with this queuedMsg.
and HandleStop() should also clear the s.QueuedMsg.


## 2026-10-08 DONE. Update Adapter interface to have `CleanUp(*Session)` fn
we want to not hog resource, cmd, stdin and stdout is exp
Cleanup() will unset all these. should be trigger by main loop only
after some X secs have passed of no activity

## 2026-10-07 DONE. update app.sessions to always hold pointers
we want to have a ez way to make sure everything is by reference and not by object.
this means that we store pointers on app.sessions instead of the actual obj
actual obj is implicitly stored and created when we "init" the session.
we choose to do it implicitly because it makes the pointer pattern easier to use, no need for copying

## 2026-10-07 DONE. never send adapter fn call directly. it can only be done in the main poll loop
we want to make sure there is only 1 place that a session is actually touched. this means we dunnid to deal with pesky mu in the connections
this means that by design, we should never call the adapter fn directly, but store enough states on the Session obj such that
in the loop, we have all the data we need to call the fn (like Msg for SendPrompt)
maybe we will have exceptions? like stop?
updated README.md

## 2026-10-07 DONE. introduction of SessionLite vs Session
we want to have the actual session object stored in main, but need the adapter.ListSessions() to be the one that pull data out
so we have the SessionLite that oni stores metadata, then upgrade it to a full session when it reaches "main space"

## 2026-10-05 DONE. all params should be in settings json
we want to have one standard place for all the configuration to sit. not some in the cli args, some in settings file.
Here, we choose to store everything in setings file. It is storage backed by default, so has consistency across sessions.

## 2026-10-01 DONE. clean design for busy state.
We want to have consistent busy state. here is design draft
Sess polling. Logic for restart.

change such that each sess now just have 2 busy flags.
- busyMsg is the busy flag based on event history. Curr logic is when (last is assistant and last_timestamp >10s) or (last_timestamp >60s). Implementation is no adapter specific
- busyCli is the busy flag based on CLI state, this will be set to true if the CLI session is in pending, compacting, etc etc basically if the session is running something. Implementation is adapter specific.

the session struct should be cross-adapter
there should be a `pollSession(s *session)` in main.go and a adapter specific one in `LoadEvents(s *session, doFullHistory bool)`

in the loop, the pollSession will be called for each session
which will trigger the adapter specific LoadEvents.
the loadEvents will populate the session.history and busyCli flag of the session.

- busyMsg should only be updated at the end of pollSession. it should oni use the information stored into the session struct
- busyCli should only ever be updated at end of loadEvents
- session.emptyCnt store the number of polls in which the len(events) doesnt change, and should be updated at end of pollSession

also updated api endpoints:
- /events dun process -N now.
- it stills process -1, when that is sent, set s.doFullHistory=true and s.busyCli=true
- events returned from /events will now have seq, which is idx of events and very first event has seq=0

rough shape
```
func pollSession(s *session) {
    cnt0 := len(s.events)

    doFullHistory = s.doFullHistory || s.emptyCnt > conf.NumEmptyPollBeforeCLIReset
    if session.harness == "pi" { loadEvents(s, doFullHistory) }
    if cnt0==len(s.events) { s.emptyCnt++ }

    if doFullHistory { s.sendFullHistory = false; s.emptyCnt = 0; }
    ... set s.busyMsg from s.events ...
}
func webHandlerEvents(startFrom int, ...) {
    s = ... get correct session ...
    if startFrom == -1 {
        s.doFullHistory = true
        s.busyCli = true
        startFrom = 0
    }

    return { events: s.events[startFrom:], busy: s.busyMsg || s.busyCli, ... }
}

// adapter specific
func piLoadEvents(s *session, doFullHistory) {
    s.busyCli = false
    if s.cli && ... some pi specific checks ... { s.busyCli = true; }
    if !s.busyCli && doFullHistory { ... code to restart cli session ... }

    if (doFullHistory) { s.events = ... full history ... }
    else {
        newEvents = ... partial poll from len(s.events)-2 ...
        ... update the s.events ...
    }
    ... update other s.vars ...

    if s.cli && ... some pi specific checks ... { s.busyCli = true; }
}

// on the ui
handleEvents(...) {
    ... if empty then skip ...
    if evts[0]["seq"]==0: CACHE[session]["events"] = []
    for evt in evts:
        CACHE[session]["events"][idx] = evt
}
```


## 2026-10-01 DONE. making gocode poll sessions parallelly
We want to speed up the wall time spent on getting the state ready on startup. At the cost of having lower level orchaestration code, we want to have control over them amt of resources we use. Therefore we will have poll loop populate the work queue, which is picked up but the list of workers. Instead of "event based style", where one session is one goroutine.

some design notes: we dunnid to have too many additional mutex, just use existing one if possible.
we only allow one big loop at one time, so no need to dedup poll attempts.
BUT /events still sometimes trigger the poll, we will fix that later.

## 2026-09-28 DONE. reduce actual calls to harnesses. make /events always return cached only. except <0 ones
we want to reduce the number of calls made to the harness. we do that by moving the "poll loop" into the gocode, (and therefore all /events calls will just read from gocode cache). We will only have 1 "actor" that calls the harness, but will pay in slight complexity in gocode.

## 2026-09-24 DONE. queue msgs
we want to be able to queue msgs. behavior: when prompt come in, we should do /events, with startFrom, if its not busy, then we send. if its busy, we add into queue. /events will know return a new val, called queue. new msgs are added on with /n/n in between.

## 2026-09-24 DONE. /events allow force with actual msg cnt
we want to force restart the viewer but dun want the full history. so change in convention. <0 means that want to forceRestart. and we will always use the "partial history" mode. except -1 which will read full history.

## 2026-09-24 storing settings in the api
we want to be able to store settings for across refreshes, like model used. therefore building generic set / get key-value datastore that can fit that fn. endpoint will be like /settings?key= if GET will be like returning it, then POST will be setting it. always return string, empty string means nothing.

save across restarts by saving to file.

## 2026-09-24 DONE. ui. add lucide for icons
we want to make site more compact, using icon pack, lucide is light choice, svg, with js helper, high level of control for syling with css

## 2026-09-24 DONE. make /events start_from=-1 means force the go code to pull full history
we want to be able to force pull history. this is useful when the state of the session get stuck in a weird state.

## 2026-09-23 DONE. /list returns harness instead of event count
we want to speed up /list. remove the per-session get_messages RPC call and return the static harness name `pi` plus the latest event timestamp, falling back to the file modification timestamp.

## 2026-09-23 DONE. functionality to swap model
we want to control the model we use. this needs changes to the api, we need to add a /models endpoint that returns list of available model, and the ability to pass that into /prompt.

## 2026-09-23 DONE. rendering markdown in ui for assistance text msg only
we want to improve scanability of replies. but we dun wan markdown to be applied acrossed. so we selectively apply it in the UI only for assistant text msgs. it should be a pure frontend behavior, with as light a markdown renderer library as possible.

we choose micromark (small + keep raw HTML disabled by default) + micromark-extension-gfm (for tables, task lists, strikethrough, and GFM links)

## 2026-09-23 DONE. /events to have a new busy flag
this is to indicate the "harness is working". best if the source is from rpc, else can infer busy=false if "last msg is assistant out" + "last msg is >10s ago". this is to be used in ui to show state, and control api flow.

## 2026-09-22 DONE. using standardized styling
we choose pico.css over other options because of its "default style" (so we dun nid to include much styling ourselves) + small size + have sufficient features for layout + styling (all that we need)

## 2026-09-22 DONE. API design + rough shape of frontend code
All endpoints should have the folder param, those that target specific session should have sessid param

/list should return the full list of sessions
/events return list of events. If not yet loaded, should go poll all previous messages. Optionally should have a "start_from" param so we dun need to always pull from history start.
/prompt to send prompt to session
/stop to stop the session

Expected ui: if I click on existing chat, the events will be called with 0, then all history will be pulled and displayed

Expected code shape on ui.
There will be a loadSessions(folder) that will populate the metadata portion + list of sessions
loadSession(folder, sessid) will pull and populate the chat.
Best use htmx for clean interaction

Always cache results locally (some design to do here - can be either indexdb or local var CACHE, but let's start with local var for simplicity) when pulled
so the load fns will follow the flow of "render from cache, then go pull and update data source, the do render after updated data"
Best if those ui components have some .cached class on the respective container that show visual hint that is cached info, like diff color or border or etcetc
Then after the full load the class can be removed

as far as possible, use rpc-ish mode, so that we dunnid to parse raw info ourselves
