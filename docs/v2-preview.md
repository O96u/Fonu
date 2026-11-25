# Fonu 2.0 preview

The v2-only FPK preview workflow builds amd64 and arm64 artifacts without publishing releases. Package version 2.0.0 is for internal testing, not a stable release. The app ID and data directory are unchanged; installing a preview can replace the existing Fonu installation. Back up configuration/data before testing, preferably on a separate fnOS instance. CI is not a substitute for browser or fnOS regression tests.

Acceptance: check initial open, narrow and maximized layout, all five accent colors and light/dark/system modes, DDNS, certificate wizard, proxy editing, FRP, logs, settings, and dashboard time filters. Compare cold and warm startup timings via the browser performance entries and network panel.
