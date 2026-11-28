# WebUI CSS Rules

## CSS-001: keep bulk CSS in web/index.html, grouped and compact

Trigger: keep bulk CSS in "web/index.html", not scattered across JS or extra stylesheets -> group under section headers, sort logically, combine related properties onto single lines.
Verify: check new styles sit in "web/index.html" under right section header without multi-line bloat.
Severity: P3 if styles scattered or verbose.

## CSS-002: prefer Pico.css and Pico CSS variables over custom styling

Trigger: lean on "pico.css" defaults and Pico CSS variables ("https://picocss.com/docs/css-variables#all-css-variables") for colors/theming -> avoid custom CSS or hardcoded values.
Verify: check new UI elements use Pico semantics/variables before adding custom rules.
Severity: P3 if custom CSS duplicates Pico or hardcodes theme colors.
