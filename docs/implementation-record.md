# Implementation and verification record

The design and plan were approved in the conversation on October 5, 2026. Execution used the native approach with independent final reviews.

- GuildSync profile controls: POC PR 86, based on voice-mute PR 85. Role tests passed 3/3; web and desktop frontend builds passed; independent review found no actionable issues.
- Companion authentication/settings: implemented separate application storage, one-use OAuth state, bounded callback listener and HTTP requests, approved-account verification, restrictive session writes and logout cleanup.
- Companion transport: six frontend tests passed, covering heartbeat/release, disconnect without replay, disable/rearm, denied requests and obsolete authentication verification.
- Native listeners: Windows production Wails build passed. Linux and macOS CGO-disabled integration builds passed; these do not verify their native listeners. Native CI builds and real desktop/Discord checks are required before claiming those platforms have been tested.
- Independent companion review found four issues: first hold ignored, stale event after disable, stale verification after logout, and native permission failure without status. All were corrected; regression coverage and the full nine-test Go race suite passed. Go vet passed.
- Frontend dependency lockfile is isolated, with Vite 8.3.2 and socket.io-client 4.8.1. Fresh dependency installation and audit reported zero vulnerabilities.
- Tagged native build/release workflow and user/platform/security documentation are included. No Google Apps Script changes were made.

## Rulings made during execution

- Use native global key-state polling every 25 milliseconds on Windows and macOS rather than installing an event hook/tap. This supports the requested hold behavior without an OS callback that needs lifecycle management or suppresses keys. Cost: an exceptionally short combination lasting less than one polling interval can be missed; this feature is intended for held shortcuts. Wayland uses the portal's activation/deactivation events.
- Use current Vite 8.3.2 in the standalone project rather than GuildSync's older Vite dependency. The first reused lockfile referenced the existing client's dependencies and was discarded; the standalone installation is independent. Cost: source builds require a current supported Node version; Node 24 is documented and used in CI. Existing GuildSync dependencies were preserved.
- Bootstrap the new public repository with a README-only master branch, then submit all application files through a feature branch and PR. Cost: the app cannot be built from the default branch until the user merges that PR.

## Press-to-mute redesign (November 2026)

- The companion now mirrors GuildSync2's voice hotkey: one registered system shortcut starts a mute, a second press restarts the countdown, and a timer always ends the mute. GuildSync2's separate unmute shortcut was not ported because it was not part of the request.
- The per-platform key-state listeners (Windows and macOS polling, X11 polling, Wayland portal activation events) and the hold-state machine were removed, along with the X11 and D-Bus dependencies. Wails v3 `GlobalShortcut` registration replaces them, and the macOS Input Monitoring requirement disappears with them.
- Settings gained `timeout_seconds` (3–120, default 10). Accelerator validation now accepts one ordinary key with optional Ctrl/Alt/Shift, which is what an operating-system accelerator can express.
- Verification here: `go build`, `go vet` and `go test` with `-tags purego`, `gofmt`, frontend tests and production build, and regenerated bindings. Native shortcut registration was not exercised on a real desktop.

## Port and follow-up work (November 2026)

- The companion runs on Wails v3.1.0 (`main.go`, `app.go`, `auth.go`), with the build system, release workflow and version stamping moved to `build/config.yml` as the single source of truth. Generated bindings are committed and a PR check regenerates them, failing on any difference.
- Custom events are registered with their payload types (`events.go`), so a mismatched payload is rejected by the event manager instead of reaching the UI silently.
- A single-instance guard (`me.perdues.guildsync.voicemute`) focuses the running window instead of letting a second copy contend for the OAuth callback port and duplicate mute requests.
- A tray icon with a mute indicator, close-to-tray and a Quit entry was added; desktops without a tray host keep close-to-exit so the app cannot become unreachable. The tray uses Wails' attach-window defaults (click toggles the window, right click opens the menu) and shows mute state in the tooltip plus a macOS status label, rather than duplicating it in a menu item. Start-at-login goes through Wails' autostart manager, which registers the platform's own login entry.
- The update check compares the embedded version with the newest published release and only opens validated GitHub release URLs; it never downloads or executes an installer.
- The embedded asset server keeps a deliberately narrow content security policy (`frame-ancestors`, `object-src`, `form-action`); `default-src`, `script-src`, `style-src` and `connect-src` are left out because the asset origin is not a plain HTTP origin on every platform (some load the bundle over the `wails://` scheme) and an untestable policy that fails closed blanks the window. Restricting script and connect origins needs a check in a real webview first, and the app renders no remote content.
- Decision: keep the generated bindings committed (they are reviewable and the drift check is cheap). Decision: do not add TypeScript bindings while the frontend is plain JS with JSDoc; the generator's TypeScript output has no consumer yet.
- Release builds are ad-hoc signed on macOS. Notarization and a Windows signing certificate need credentials this repository does not have.

## Remaining verification limits

Live Discord login, account approval, actual channel mutes, native shortcut registration (including Wayland portal binding), the auto-unmute timer, tray behavior, start-at-login entries and the update check have not been exercised against production accounts or real desktops here. The release matrix has only been exercised locally on Linux with `-tags purego`; a dispatchable dry-run build was added so the Windows and macOS packaging paths can be proven without publishing. The bot's existing permissions, durable state and expiry remain authoritative. A successful build is not an end-to-end Discord test.
