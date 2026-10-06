# GuildSync Voice Mute

A small desktop companion for GuildSync's Discord voice-channel mute feature. It signs in with Discord and uses the existing GuildSync backend and bot; you do not run a second Discord bot.

## Use

1. Open the application and choose **Sign in with Discord**. Complete login in your browser. Your GuildSync account must be approved as **User** or **Admin**, and your Discord role must be allowed by the server's voice-mute policy.
2. Choose **Set shortcut**, then press one or more supported keys and release all of them to save. Examples: `F8`, `Space`, `M+N`, or `Ctrl+Shift+M`. Letters, numbers, F1–F12, Ctrl/Alt/Shift, and navigation keys are supported. Escape cancels. **Return to Default** restores Ctrl+M and disables the shortcut.
3. Enable the global shortcut and join a Discord voice channel. Hold the shortcut to request temporary mutes for eligible lower-ranked members in that channel. Release it to end the request.

Moderator mutes are preserved. Equal/higher and unknown guild ranks, bots and the server owner are protected by the existing bot. Changing channels ends the request; it does not follow you into another channel. Disconnect, logout, shortcut changes and app shutdown stop the request. On reconnect, release the keys and press again.

The application must remain running. Closing its window exits; this first version does not have a system tray. Choosing a shortcut that conflicts with ESO or another application may trigger both actions, because ordinary key input is not suppressed.

## Server requirements

Deploy [GuildSync's voice-mute feature](https://github.com/skperdue68/POC/pull/85) and configure its allowed Discord role names or IDs, rank order, and enabled policy. The bot needs **Mute Members** and **View Audit Log**. The companion connects to `https://guildsync.perdues.me`, using the same authenticated endpoint as GuildSync. There is no port 3005 companion service.

The Enable checkbox remains disabled until you sign in with an approved User/Admin account. Account approval and the role allowlist are separate requirements. A new account can be pending or a Viewer until an administrator grants the required access. GuildSync Admin status does not bypass Discord rank checks.

## Install and platform support

Tagged builds publish Windows, macOS and Linux archives. Extract the platform archive, then run the executable/application. Windows requires WebView2. macOS builds are unsigned; distribution signing/notarization is not included. Linux builds target Ubuntu 24.04-compatible GTK3/WebKitGTK 4.1 environments.

See [platform support and permissions](docs/platform-support.md), including macOS Input Monitoring and Linux Wayland portal requirements. Native runners build all three platforms; actual global-shortcut behavior still needs verification on a real desktop. A successful build alone is not a claim that every desktop environment works.

## Saved login and settings

The companion uses its own `GuildSync-Voice-Mute` directory in your OS's user configuration folder. It saves `session.json` and `settings.json`; it does not change GuildSync's existing saved login. Do not share the session file: it contains your account bearer token. Sign out removes it locally and attempts server logout. GuildSync sessions can be revocable without a fixed expiry timestamp; the companion accepts this format and still verifies saved sessions with the backend. A supplied expiry is enforced.

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

CI runs tests and produces executable archives. Release tags must match the version in `wails.json` and `frontend/package.json`; mismatches fail rather than creating misleading filenames. Product changes are reviewed through pull requests.
