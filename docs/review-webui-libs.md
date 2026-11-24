# WebUI Library & Asset Rules

## LIB-001: load all third-party assets locally from web/lib

Trigger: All CSS/JS library files must be vendored and loaded from `web/lib/`, never from remote CDNs (e.g. `unpkg`).

Verify: Search `web/` for `http://` or `https://` `<script>`/`<link>` tags and confirm none point to external hosts.

Severity: P1 if the UI depends on remote assets that break offline or local-only usage.

## LIB-002: use Lucide icons instead of verbose labels

Trigger: Use `lucide` icons wherever appropriate to replace words and keep the UI compact.

Verify: Check new buttons/controls use Lucide icons where an icon cleanly conveys the action.

Severity: P3 if text-heavy controls clutter compact UI areas.
