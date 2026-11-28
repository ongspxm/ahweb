# WebUI Interaction & Feedback Rules

## UX-001: order action dialog and hotkeys by semantics, not keybindings

Trigger: group and order actions (including hotkey-less ones) by action semantics, never shortcut letter.
Verify: inspect action lists/dialogs -> grouping matches what actions do, not key order.
Severity: P3 if actions feel arbitrarily ordered.

## UX-002: avoid Shift-dependent hotkeys

Trigger: shifted keys fail capture (e.g. "leader + ?" fails because "?" is "Shift + /") -> use unshifted keys only.
Verify: check "web/app_hotkey.js" bindings -> unshifted keys only.
Severity: P2 if bound or documented hotkey unreachable.

## UX-003: show notifications and errors in notice-area

Trigger: route user info, notifications, and errors to "notice-area".
Verify: check user-facing alerts/errors render via "notice-area", not "alert()", console-only logs, or ad-hoc banners.
Severity: P2 if errors or key feedback stay hidden.
