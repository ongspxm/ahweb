# WebUI Interaction & Feedback Rules

## UX-001: order action dialog and hotkeys by semantics, not keybindings

Trigger: Always group and order actions (including ones without hotkeys) by logical/semantic grouping, never by keyboard shortcut order.

Verify: Inspect action lists/dialogs and confirm grouping reflects what actions do, not their key letters.

Severity: P3 if actions feel arbitrarily ordered.

## UX-002: avoid Shift-dependent hotkeys

Trigger: we want clear hotkeys definition
shifted keys may not captured properly (e.g. `leader + ?` fails because `?` is `Shift + /`).

Verify: Check new keybindings in `web/app_hotkey.js` use unshifted keys only.

Severity: P2 if a documented/bound hotkey is unreachable.

## UX-003: show notifications and errors in notice-area

Trigger: Always use `notice-area` for information, notifications, or errors that need to be highlighted to the user.

Verify: Check new user-facing alerts/errors render via `notice-area` rather than `alert()`, console-only logs, or ad-hoc banners.

Severity: P2 if errors or important feedback are hidden from the user.
