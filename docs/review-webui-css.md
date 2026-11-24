# WebUI CSS Rules

## CSS-001: keep bulk CSS in web/index.html, grouped and compact

Trigger: Keep styling centralized in `web/index.html` instead of scattering styles across JS or extra stylesheets.
Group declarations under section headers, sort by logical groupings, and combine related properties onto single lines to minimize declarations.

Verify: Check new styles live in `web/index.html` under the right section header and don't bloat into multi-line boilerplate.

Severity: P3 if styles are scattered or unnecessarily verbose.

## CSS-002: prefer Pico.css and Pico CSS variables over custom styling

Trigger: Minimize custom CSS by leaning on `pico.css` defaults and styling.
Use Pico CSS variables (`https://picocss.com/docs/css-variables#all-css-variables`) for colors and theming instead of hardcoded values.

Verify: Check new UI elements use Pico semantics/variables before adding custom CSS rules.

Severity: P3 if custom CSS duplicates Pico styling or hardcodes theme colors.
