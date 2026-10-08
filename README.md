# GuildSync Voice Mute

A small desktop companion for GuildSync's Discord voice-channel mute feature. It signs in with Discord and uses the existing GuildSync backend and bot; you do not run a second Discord bot.

## Use

1. Open the application and choose **Sign in with Discord**. Complete login in your browser. You do not need a GuildSync account or GuildSync approval. You must belong to the configured Discord server and have a role allowed by its voice-mute policy.
2. Choose **Set shortcut**, then press one or more supported keys and release all of them to save. Examples: `F8`, `Space`, `M+N`, or `Ctrl+Shift+M`. Letters, numbers, F1–F12, Ctrl/Alt/Shift, and navigation keys are supported. Escape cancels. **Return to Default** restores Ctrl+M and disables the shortcut.
3. Enable the global shortcut and join a Discord voice channel. Hold the shortcut to request temporary mutes for eligible lower-ranked members in that channel. Release it to end the request.

Moderator mutes are preserved. Equal/higher and unknown guild ranks, bots and the server owner are protected by the existing bot. Changing channels ends the request; it does not follow you into another channel. Disconnect, logout, shortcut changes and app shutdown stop the request. On reconnect, release the keys and press again.

The application must remain running. Closing its window exits; this first version does not have a system tray. Choosing a shortcut that conflicts with ESO or another application may trigger both actions, because ordinary key input is not suppressed.

## Server requirements

Deploy [GuildSync's voice-mute feature](https://github.com/skperdue68/POC/pull/85) and configure its allowed Discord role names or IDs, rank order, and enabled policy. The bot needs **Mute Members** and **View Audit Log**. The companion connects to `https://guildsync.perdues.me`, using dedicated voice-only authentication endpoints on the existing backend. There is no port 3005 companion service.

The Enable checkbox remains disabled until Discord login succeeds. Discord membership, permitted requester roles, recognized ranks and bot permissions remain authoritative for mute requests. GuildSync Viewers may also mute from GuildSync when their Discord role permits it; a GuildSync Admin role does not bypass Discord rank protection.

## Install and platform support

Published GitHub releases build and publish Windows, macOS and Linux archives. Extract the platform archive, then run the executable/application. Windows requires WebView2. macOS builds are unsigned; distribution signing/notarization is not included. Linux builds target Ubuntu 24.04-compatible GTK3/WebKitGTK 4.1 environments.

See [platform support and permissions](docs/platform-support.md), including macOS Input Monitoring and Linux Wayland portal requirements. Native runners build all three platforms; actual global-shortcut behavior still needs verification on a real desktop. A successful build alone is not a claim that every desktop environment works.

## Saved login and settings

The companion uses its own `GuildSync-Voice-Mute` directory in your OS's user configuration folder. It saves `session.json` and `settings.json`; it does not change GuildSync's existing saved login. Do not share the session file: it contains your account bearer token. Sign out removes it locally and attempts server logout. New standalone voice sessions expire after 30 days, can be revoked on sign-out, and are verified by the backend. Existing saved GuildSync logins must be replaced by signing in again after this update. Standalone tokens authorize only mute operations, not GuildSync data.

Backend startup creates separate `guildsync_voice_identities` and `guildsync_voice_login_sessions` tables. Discord identities stored there do not appear in GuildSync account management. Later logging into GuildSync itself creates the usual account and follows its normal approval process. Deploy the matching GuildSync backend PR before installing this client.

The shared browser callback listens at `127.0.0.1:53682` during login. If another GuildSync login is already listening, finish or close that login and try again. This is a local browser callback, not a new server port.

## Build from source

Install Go 1.23 or newer, Node.js 24, and Wails v2.12.0 plus its native platform dependencies.

```sh
go install github.com/wailsapp/wails/v2/cmd/wails@v2.12.0
cd frontend
npm ci
npm test
npm run build
cd ..
go test ./...
wails build
```

On Ubuntu 24.04, install `build-essential pkg-config libgtk-3-dev libwebkit2gtk-4.1-dev libx11-dev` and build with `wails build -tags webkit2_41`. macOS needs Xcode Command Line Tools. Development builds use `wails dev`.

Pull requests run tests only. Ordinary pushes do not build the application. Publish a GitHub release with a tag such as `v1.0.1` to build Windows, macOS and Linux archives and attach them to that release. The release tag stamps `wails.json`, frontend package metadata, archive filenames and the visible version before building. The version appears in the window title and below the app heading. Local source builds use the version in `wails.json`; run `python tools/set-release-version.py v1.0.1` to change it locally. Product changes are reviewed through pull requests.

