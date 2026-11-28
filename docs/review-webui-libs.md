# WebUI Library & Asset Rules

## LIB-001: load all third-party assets locally from web/lib

Trigger: vendor all CSS/JS libs in "web/lib/" -> never load from remote CDNs (e.g. "unpkg").
Verify: search "web/" for "http://" or "https://" in "<script>"/"<link>" tags -> zero external hosts.
Severity: P1 if UI depends on remote assets and breaks offline/local-only usage.

## LIB-002: use Lucide icons instead of verbose labels

Trigger: use "lucide" icons over wordy labels -> keeps compact UI clean.
Verify: check new buttons/controls use Lucide icons where icon clearly conveys action.
Severity: P3 if text-heavy controls clutter compact UI.
