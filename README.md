# GuildSync Voice Mute

A small desktop companion for GuildSync's Discord voice-channel mute feature. It signs in with Discord and uses the existing GuildSync backend and bot; you do not run a second Discord bot.

## Use

1. Open the application and choose **Sign in with Discord**. Complete login in your browser. You do not need a GuildSync account or GuildSync approval. You must belong to the configured Discord server and hold a role allowed by its voice-mute policy (see [Server requirements](#server-requirements)).
2. Choose **Set shortcut**, then press one key, optionally with Ctrl, Alt or Shift, and release it to save. Examples: `F8`, `Space`, `Ctrl+Shift+M`, or `Alt+F2`. Letters, numbers, F1–F12, Ctrl/Alt/Shift, and navigation keys are supported. Escape cancels. **Return to Default** restores Ctrl+M, the 10-second auto-unmute and disables the shortcut.
3. Enable the mute shortcut and join a Discord voice channel. Press the shortcut to mute eligible lower-ranked members in that channel for the configured time (10 seconds by default; set 3–120 seconds in **Auto-unmute after**). Pressing it again restarts the countdown instead of unmuting, and every mute ends on its own.

Moderator mutes are preserved. Equal/higher and unknown guild ranks, bots and the server owner are protected by the existing bot. Changing channels ends the request; it does not follow you into another channel. The auto-unmute timer, disconnect, logout, shortcut changes and app shutdown stop the request. On reconnect, press the shortcut again.

The companion must remain running while you use the shortcut. On desktops with a system tray icon it keeps running after you close the window and exits from the tray menu; where the desktop has no tray host, closing the window exits. Enable **Start at login** to launch it with your computer. Choosing a shortcut that conflicts with ESO or another application may trigger both actions, because ordinary key input is not suppressed.

## Server requirements

Deploy [GuildSync's voice-mute feature](https://github.com/skperdue68/POC/pull/85) and configure it on the GuildSync side; nothing in this companion decides who may mute. Access is **Discord-role based, with no per-account allowlist**: a member is allowed because of the Discord roles they hold, never because their account was listed individually. The bot needs **Mute Members** and **View Audit Log**. The companion connects to `https://guildsync.perdues.me`, using dedicated voice-only authentication endpoints on the existing backend. There is no port 3005 companion service.

The bot reads three settings, set in `NodeJS/GuildSync-Discord-Bot/.env` in the GuildSync repository:

| Setting | Default | Meaning |
| --- | --- | --- |
| `GUILDSYNC_VOICE_MUTE_ENABLED` | `false` | Whether new mute requests are accepted at all. Disabling it ends live sessions and clears known temporary mutes. |
| `GUILDSYNC_VOICE_MUTE_ALLOWED_ROLE_IDS` | blank | Comma-separated **Discord role names or numeric IDs** allowed to request mutes, for example `Kingpin, Consigliere`. Names are case-insensitive and omit `@`. Blank allows nobody. |
| `GUILDSYNC_VOICE_MUTE_RANK_ROLE_IDS` | blank | Guild-rank roles from lowest to highest, as names or numeric IDs. Blank discovers the recognized ranks (Gangsters, Associates, Soldiers, Capos, Caporegime, Consigliere, Kingpin). Only strictly lower ranks can be muted. |

The `_ROLE_IDS` suffix is historical; role names work in both settings. The same three settings can be changed live in GuildSync under **Reports and Admin → Administrator Configuration → Voice channel mute** and saved as server settings, which override the bot's environment defaults without a restart. If an account is refused, check that screen first, then the bot's `.env`.

The Enable checkbox remains disabled until Discord login succeeds. Discord membership, permitted requester roles, recognized ranks and bot permissions remain authoritative for mute requests; each request is rechecked when it is made. GuildSync Viewers may also mute from GuildSync when their Discord role permits it; a GuildSync Admin role does not bypass Discord rank protection.

## Install and platform support

Published GitHub releases attach an installer for each platform next to the portable archives: `GuildSync-Voice-Mute-<version>-Windows-Setup.exe` (Inno Setup, per-user, no elevation), `GuildSync-Voice-Mute-<version>-macOS.dmg` (open it and drag the app into Applications) and `GuildSync-Voice-Mute-<version>-Linux-<arch>.AppImage` (such as `-Linux-x86_64.AppImage`), which runs on any distribution. Each installer is published as the installer itself, with no zip around it, so a download is ready to run and that same file is what the in-app update fetches. A release is only published once all three installers exist. The portable archives stay available: extract the zip, then run the executable/application. Windows requires WebView2. macOS builds are ad-hoc signed but not notarized, so the first launch needs **Open Anyway** in System Settings → Privacy & Security. The Linux AppImage bundles the GTK4/WebKitGTK 6.0 runtime, so it needs to be made executable and nothing else; the Linux zip does not bundle it and needs `libgtk-4-1` and `libwebkitgtk-6.0-4` or the equivalent packages installed.

**Start at login** stores the exact path of the executable you enabled it from, so a portable copy has to stay where it was extracted: moving, renaming or deleting it afterwards leaves the login entry pointing at the old location, and the checkbox reads off until you enable it again from the new path. The installers put the app in a fixed location, so they are not affected. When the login item cannot be applied at all — an unsupported desktop, or macOS still waiting for approval — the checkbox greys out and explains why underneath instead of silently switching itself back off. On macOS, keep the `.app` in a permanent location such as `/Applications` and approve it under **System Settings → General → Login Items** (**Login Items & Extensions** on macOS 15 and later); until that approval is given, macOS reports the registration as not enabled. The entry takes effect at the next login and launches the app with its window, so it is not started hidden.

See [platform support and permissions](docs/platform-support.md), including Linux Wayland portal requirements. The shortcut is registered through the operating system, so no keyboard listener or macOS Input Monitoring permission is involved. Native runners build all three platforms; actual global-shortcut behavior still needs verification on a real desktop. A successful build alone is not a claim that every desktop environment works.

## Updates

Choose **Check for updates** to compare this build with the newest published GitHub release. The check is user-initiated and only contacts the GitHub releases API. When a newer version exists, the companion downloads that release's installer **for the platform you are running** — `...-Windows-Setup.exe`, `...-macOS.dmg` or the `...-Linux-<arch>.AppImage` built for your machine — into your Downloads folder, checks it against the SHA-256 digest GitHub publishes for the file, and opens it. There is no archive to unpack. The Linux AppImage is matched by architecture, so an arm64 machine asks for the `aarch64` image instead of one it cannot run; when a release only carries another architecture, the status line says so and the download page stays available. Quit the companion before finishing the install and start the new version afterwards. **Open download page** stays available and opens only that release page on GitHub, so it remains the fallback when a release has no installer for your platform or the download fails; the status line says which happened. If you installed Linux from v2.0.5 — the last release that shipped a `.deb` — that build still looks for a `.deb`, so it shows the newer version without a download and sends you to the release page: download the AppImage there, make it executable and re-enable **Start at login** from it. The saved login and shortcut settings carry over, because both builds use the same user configuration directory.

## Saved login and settings

The companion uses its own `GuildSync-Voice-Mute` directory in your OS's user configuration folder. It saves `session.json` and `settings.json` (shortcut, enabled state and auto-unmute duration); it does not change GuildSync's existing saved login. Do not share the session file: it contains your account bearer token. Sign out removes it locally and attempts server logout. New standalone voice sessions expire after 30 days, can be revoked on sign-out, and are verified by the backend. Existing saved GuildSync logins must be replaced by signing in again after this update. Standalone tokens authorize only mute operations, not GuildSync data.

Backend startup creates separate `guildsync_voice_identities` and `guildsync_voice_login_sessions` tables. Discord identities stored there do not appear in GuildSync account management. Later logging into GuildSync itself creates the usual account and follows its normal approval process. Deploy the matching GuildSync backend PR before installing this client.

The shared browser callback listens at `127.0.0.1:53682` during login. If another GuildSync login is already listening, finish or close that login and try again. This is a local browser callback, not a new server port.

## Build from source

Install Go 1.27.3 or newer, Node.js 24, and the Wails v3.1.0 CLI plus its native platform dependencies.

```sh
go install github.com/wailsapp/wails/v3/cmd/wails3@v3.1.0
cd frontend
npm ci
npm test
npm run build
cd ..
go test ./...
wails3 build
```

On Ubuntu 24.04, install `build-essential pkg-config libgtk-4-dev libwebkitgtk-6.0-dev libx11-dev`. macOS needs Xcode Command Line Tools. Development builds use `wails3 dev`. Go methods exposed to the window are called through generated bindings in `frontend/bindings`; run `wails3 generate bindings` after changing exported methods.

Pull requests run tests only. The Linux compile job also bundles the application, builds the AppImage and checks the copy extracted from its zip; a smoke job does the same for the Windows and macOS installers on their native runners, installing the setup silently and mounting the DMG. Ordinary pushes do not build the application. Publish a GitHub release with a tag such as `v1.0.1` to build Windows, macOS and Linux archives and attach them to that release. The release tag stamps `build/config.yml`, the Windows and macOS bundle metadata, frontend package metadata, archive filenames and the visible version before building. The build output lands in `bin/`. The version appears in the window title and below the app heading. Local source builds use the version in `build/config.yml`; run `python tools/set-release-version.py v1.0.1` to change it locally. Product changes are reviewed through pull requests.

