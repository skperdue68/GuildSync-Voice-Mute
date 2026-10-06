# Voice Companion Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox syntax for tracking.

**Goal:** Deliver the requested GuildSync profile controls and a public, authenticated GuildSync-Voice-Mute desktop companion for Windows, macOS and Linux.

**Architecture:** Keep authorization and mute ownership in the existing GuildSync backend/bot. A small Go/Wails application supplies native shortcut edges and an authenticated Socket.IO connection. Platform listeners, local settings, OAuth and the frontend session controller remain independent units.

**Tech Stack:** Go 1.23+, Wails v2.12, Vite, socket.io-client v4, native Windows/macOS/X11 listeners, and D-Bus GlobalShortcuts portal on supported Wayland desktops.

**Spec:** ../specs/2026-10-05-standalone-voice-mute-design.md

## Global Constraints

- Repository: public `GuildSync-Voice-Mute`; product changes use branches and pull requests.
- Leave the primary POC checkout on master and preserve its uncommitted files.
- Use the existing GuildSync authenticated endpoint, default `https://guildsync.perdues.me`; no separate port 3005 listener.
- Local settings default to disabled and `Ctrl+M`; keep companion application data separate from GuildSync.
- Never ship bot tokens, Discord client secrets, database credentials or live saved sessions.
- Global shortcut support must report denied permission or unavailable desktop support honestly.
- Existing bot channel, rank and moderation protections remain authoritative.
- No Google Apps Script changes are required.

## Review Focus

- Another application occupies the OAuth callback port: show a useful error without changing the accepted redirect.
- Login is revoked while the app remains open: end local activity and reject subsequent presses.
- The app starts or reconnects while keys are held: wait for release and a new press.
- Wayland portal permission is declined or its backend lacks support: remain disabled with a useful status.
- Closing the window or changing shortcuts during a mute: cancel heartbeats and attempt release; server expiry handles delivery failure.

## Task 1: GuildSync profile controls

**Files, relative to POC:** both `GO/GuildSync-Frontend-Client/frontend/src/` and `NodeJS/GuildSync-Backend-Server/web/src/` copies of `role-view-controls.js` and `user-administration.css`; web `main.js`; backend `role-view-controls.test.js`.

**Interfaces:** Preserve `renderRoleViewControls(user)` and `[data-role-view]` values `user`, `viewer`, `admin`. Do not alter role authorization or general Manage Users button layout.

- [ ] Create an isolated `codex/profile-view-buttons` worktree from the voice feature. Verify PR 85's state first; if still open, use its branch as the new PR base to avoid duplicating that feature.
- [ ] Update existing tests: real admins get two actions with accessible names View As User/Viewer; previews have only Return to Admin View, no preview-account notice; ordinary users/viewers receive no controls; both clients remain identical. Run `node --test role-view-controls.test.js` in the backend and confirm the changed expectations fail before implementation.
- [ ] Render two-line centered User/Viewer buttons in a two-column group; add scoped colored styles and a full-width centered return button. Use a small `aria-hidden` spacer instead of the preview-account notice. Remove only the web profile's personal Voice Channel Mute section, retaining server administrator settings.
- [ ] Run the test again and both frontend builds. Inspect the rendered controls at desktop and narrow widths, including focus and hover states. Publish the source and existing repository-required generated assets through a separate branch/PR.

## Task 2: Companion login and local persistence

**Files, relative to new repository:** `main.go`, `app.go`, `auth.go`, `auth_test.go`, `settings.go`, `settings_test.go`, `go.mod`, `wails.json`, `frontend/package.json`, `frontend/src/main.js`, `frontend/src/style.css`.

**Interfaces:** `App.StartDiscordLogin() error`, `App.GetSession() (Session, error)`, `App.Logout() error`, `App.GetShortcutSettings() (ShortcutSettings, error)`, `App.SetShortcutSettings(ShortcutSettings) error`. `Session` exposes only client-safe token, expiry, account and endpoint information; `ShortcutSettings` has `enabled` and `shortcut`.

- [ ] Prepare the application in an isolated local directory under the permitted workspace. Reuse GuildSync's public client configuration and backend desktop-token/session/logout endpoints, without copying unrelated banking/file-watch functionality.
- [ ] Add failing Go tests for OAuth state mismatch, expired/revoked session, pending approval, occupied callback port, malformed settings, and separate companion storage. Run `go test ./...` and verify the failures.
- [ ] Implement browser login with random one-use state, loopback callback, bounded HTTP timeouts and validated responses. Save sessions with restrictive permissions and atomic replacement. Stop the callback listener on completion/shutdown. Bind settings validation to the shared shortcut parser.
- [ ] Implement the minimal GUI with login, account/connection status, enable switch, capture/reset shortcut and logout. Pending/denied accounts cannot activate the shortcut. Run Go tests and `npm run build` in `frontend`; inspect the GUI without requiring production credentials.
- [ ] Commit the working authenticated shell on the feature branch.

## Task 3: Session transport and shortcut lifecycle

**Files:** `frontend/src/voice-session.js`, `frontend/src/voice-session.test.js`, `shortcut.go`, `shortcut_test.go`, `app.go`.

**Interfaces:** `NewShortcutListener(shortcut string, onEdge func(bool)) (ShortcutListener, error)`; `ShortcutListener.Close() error`. Emit Wails event `voice-shortcut-edge` with a Boolean pressed state. Export `createVoiceSession({getSocket, clock, notify})` with `onEdge(pressed)`, `stop()` and `onDisconnect()` methods.

- [ ] Add failing controller tests for press/release, two-second heartbeat, stale acknowledgments, rejection, logout, shortcut change, disconnect and reconnect while held. Assertions require no new session until release followed by press and no heartbeat after stop. Run `node --test frontend/src/voice-session.test.js`.
- [ ] Connect using the existing GuildSync Socket.IO auth payload. Send `guildsync:voice-mute-hotkey` with `{state, sessionId}` and a new random session ID per press. Never accept requester identity or channel selection from the GUI.
- [ ] Handle acknowledgement errors visibly; release and cancel timers before clearing login or replacing a listener. Keep network work outside native keyboard callbacks. Closing/shutting down attempts release; retain the bot's eight-second expiry fallback.
- [ ] Run controller and Go tests, including settings changes while held and application startup while held; commit.

## Task 4: Native platform listeners

**Files:** `shortcut_windows.go`, `shortcut_darwin.go`, `shortcut_darwin.m`, `shortcut_linux.go`, `shortcut_x11.go`, `shortcut_wayland.go`, platform-specific tests and `docs/platform-support.md`.

**Interfaces:** Implement Task 3's listener interface; expose `App.GetShortcutSupport() (ShortcutSupport, error)` where support includes `available`, `reason`, and permission state. No platform may silently fall back to a focused-window listener.

- [ ] Verify current primary platform documentation before choosing API calls/dependencies. For Linux, select Wayland when `WAYLAND_DISPLAY` is present; use X11 otherwise. Record required native libraries and portal dependencies.
- [ ] Add failing adapter tests for permission denial, missing portal backend, event deduplication, initial-held suppression and safe listener shutdown. Use injected event sources for deterministic unit tests.
- [ ] Reuse the Windows low-level hook and modifier-state sampling approach. Implement macOS session event observation with explicit input-monitoring permission/status and event-tap recovery. Implement X11 key state observation without consuming normal input. Use portal CreateSession/BindShortcuts and Activated/Deactivated on Wayland, accepting the desktop's permission/binding UI as authoritative.
- [ ] Wire shortcut replacement so the old listener closes before the new one can activate. Failed setup leaves the control disabled and displays its reason. Run Go tests and native builds on all three CI runners.
- [ ] Manually verify press/release outside the app, normal key delivery, shutdown, logout and reconnect cleanup on each available desktop. Mark untested platforms as unverified in delivery notes; commit.

## Task 5: Public repository, releases and review

**Files:** `.github/workflows/build.yml`, `.github/workflows/release.yml`, `.gitignore`, `README.md`, `docs/platform-support.md`, `docs/security-and-setup.md`.

**Interfaces:** Publish tagged release archives `GuildSync-Voice-Mute-<version>-Windows.zip`, `GuildSync-Voice-Mute-<version>-macOS.zip`, and `GuildSync-Voice-Mute-<version>-Linux.zip`; keep version inputs consistent across app metadata and artifact names.

- [ ] Build and test on native Windows/macOS/Linux GitHub Actions runners using pinned Wails and lockfiles. Install documented native build dependencies; package the actual Wails executable/app bundle, not source alone. A tag triggers release publication only after checks succeed.
- [ ] Document account approval, required Discord roles, server enablement, saved-login location, shortcut conflicts, platform permissions and recovery behavior. Clarify that server policy controls remain in GuildSync and no standalone bot is deployed.
- [ ] Inspect tracked files for credentials/session data and verify the complete local suite/build. Obtain an independent code review focused on authentication, held-key reconnect behavior and cleanup.
- [ ] Create public `skperdue68/GuildSync-Voice-Mute` using an authenticated GitHub capability without exposing credentials. Bootstrap its default branch with repository documentation only, then push the product feature branch and open a PR. Do not merge it.
- [ ] Report both PR links, verification evidence, platform limitations and deployment steps. Mention that PR 85 must be deployed for the companion's existing event protocol to work.

## Execution recommendation

Use native execution: implement the tasks in this session, followed by an independent final review. The tasks share authentication, shortcut and transport interfaces, so one implementation owner reduces integration churn. Review this plan before product implementation, as required by the writing-plans workflow.
