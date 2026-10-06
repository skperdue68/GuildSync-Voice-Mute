# GuildSync-Voice-Mute design for review

## Purpose and delivery

Create a public repository named GuildSync-Voice-Mute. Provide a small desktop companion for Windows, macOS and Linux users who do not run the full GuildSync client. It uses the existing GuildSync backend and Discord bot. Product changes are delivered through a feature branch and pull request. The existing VS Code checkout remains on master.

## Application and authentication

Use Go and Wails, consistent with GuildSync's desktop build. The window contains Sign in with Discord, account and connection status, an Enabled switch, shortcut capture, Return to Default, and Logout. Ctrl+M is the initial shortcut; the feature starts disabled. The shortcut is held to mute and released to end the session.

Reuse GuildSync's browser-based Discord OAuth and approved-account checks. Save the login and local shortcut settings in a separate GuildSync-Voice-Mute application-data directory. Never distribute a Discord client secret, bot token or database credentials. Expired or revoked sessions require login again. Logout ends the active mute before removing the saved session.

Connect to the existing authenticated GuildSync Socket.IO endpoint and existing voice-mute events. The default public endpoint is https://guildsync.perdues.me; there is no additional mutebot listener or port 3005. Reuse the currently configured loopback OAuth callback when available; explain an occupied callback port rather than silently changing a redirect that the server does not accept.

## Mute safety

All channel selection, account authorization, Discord role checks and durable moderation state remain on the existing backend and bot. The companion cannot supply a trusted requester identity or select other channels. Equal/higher ranks remain protected. Confirmed external moderator mutes are preserved.

Send press, release and periodic heartbeat events. Disconnect, logout, shutdown, disabling, or changing the shortcut ends the local session. Server expiry remains the fallback when release cannot be delivered. Reconnection never replays a held shortcut: the person must release and press again.

## Platform support

Windows uses the existing native keyboard-hook approach. macOS needs a native global event listener and an explicit permission/status flow; denied permission must leave the feature disabled. Linux needs separate X11 and Wayland handling. For Wayland, use the desktop's GlobalShortcuts portal when available, including activation and deactivation events. Unsupported desktop backends must display a clear unavailable status rather than pretending a focused-window shortcut is global. Building an executable does not establish that every desktop supports global shortcuts.

Keep platform listeners behind one press/release interface. Listeners must not block native callbacks with network operations, suppress normal key input, or start a session merely because the application launches while keys are held. If a platform cannot bind the selected combination, show the reason and require another selection.

## Builds and verification

GitHub Actions runs relevant Go/frontend tests and builds Windows, macOS and Linux executables on native runners. Tagged releases publish clearly named platform archives and installation instructions. Document macOS permissions and unsigned-app behavior, and Linux runtime dependencies and desktop limitations.

Test authentication/state handling, shortcut changes, heartbeat cancellation, reconnect without replay, and unavailable/denied platform support. Manually verify actual global press/release behavior and cleanup on each supported desktop before claiming that platform works. CI cross-compilation alone is insufficient.

## Related GuildSync profile changes

On both clients, show View As / User and View As / Viewer as two colored buttons beside each other, with centered two-line labels. The return control is a full-width colored Return to Admin View button. Replace the preview-account notice with a small blank spacer. Preserve existing role-preview permissions and event handlers.

Remove the personal hotkey configuration/unsupported notice from the web profile. Retain the server's administrator voice-mute policy settings, which configure the bot and remain useful from the web.

## Alternatives considered

A console-only companion would reuse less UI code but would not provide the requested login and shortcut GUI. An Electron companion could share JavaScript transport code, but introduces another desktop framework and a larger distribution. Go/Wails best fits the existing client and release tooling, with native platform listeners isolated from authentication and transport.

## Review boundary

This document specifies the new application and accompanying profile adjustments. The standalone repository has not been created and these profile adjustments have not yet been implemented. Existing voice-mute implementation remains in POC pull request 85. After review, an implementation plan will identify platform dependencies and delivery steps before product implementation.
