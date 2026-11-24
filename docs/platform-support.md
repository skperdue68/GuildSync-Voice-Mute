# Platform support and permissions

The shortcut is registered with the operating system through Wails v3's global shortcut API. The app is told when the accelerator is pressed; it never reads key state, installs a keyboard hook, or suppresses key input. Because nothing depends on detecting a key release, no platform input-monitoring permission is required. Muting is press-to-toggle: one press mutes for the configured time, another press restarts the countdown, and the timer always ends the mute.

## Windows

The accelerator is registered with `RegisterHotKey`, so it works while another application has focus. No keys are intercepted or suppressed. WebView2 is needed for the GUI. Secure desktops, lock screens and disconnected sessions do not provide an interactive desktop for shortcut delivery, and a shortcut Windows already reserves fails to register with an error in the app.

## macOS

The accelerator is registered as a system-wide hotkey; **System Settings → Privacy & Security → Input Monitoring** and Accessibility are not needed. macOS can reserve shortcuts or treat function keys as media controls; use an ordinary modifier/letter combination when necessary. Registration failures are reported in the app. Release builds are ad-hoc signed (`codesign --sign -`) so the bundle is internally consistent, but they are not notarized: the first launch needs the usual Gatekeeper **Open Anyway** confirmation.

## Linux X11

Requires an accessible X11 display with GTK4/WebKitGTK 6.0 (`libgtk-4-dev libwebkitgtk-6.0-dev` on Ubuntu 24.04). Wails grabs the accelerator; the app no longer queries key state, so changing the keyboard layout does not require re-enabling the shortcut.

## Linux Wayland

Requires a desktop implementing the [GlobalShortcuts portal](https://flatpak.github.io/xdg-desktop-portal/docs/doc-org.freedesktop.portal.GlobalShortcuts.html) and the appropriate `xdg-desktop-portal` backend. The desktop may show its own approval/binding dialog. Its actual binding is authoritative and may differ from the app's requested shortcut; check the dialog. Declining permission or a missing backend leaves activation unavailable. The app never falls back to a focused-window shortcut.

The portal supports one ordinary key (such as F8 or M), optionally with Ctrl/Alt/Shift. Modifier-only shortcuts and multiple ordinary keys (such as `M+N`) are rejected with an explanation; those combinations also fail on Windows, macOS and X11, because an accelerator cannot express them.

A missing GlobalShortcuts API or inaccessible desktop session does not fail installation or application startup. The app reports the registration error, disables Enable and offers **Set shortcut** to try a different combination. A failure is remembered for the shortcut that caused it, so the desktop is not prompted again until the shortcut changes. The app does not request broader permissions to compensate for a missing API.

## Installers

Each release attaches one installer per platform next to the portable zips, all three built from the same `bin/` payload by `tools/package_installers.py`, and each installer is published exactly as it was built — no zip around it — because that file is what the in-app update downloads:

- **Windows** — an Inno Setup installer (`...-Windows-Setup.exe`). It installs per user into `%LOCALAPPDATA%\Programs\GuildSync Voice Mute`, so neither the installer nor the app needs elevation, and adds a Start menu entry, an optional desktop shortcut and an entry under Apps & features. It does not bundle WebView2; Windows 10 and 11 already provide the Evergreen runtime.
- **macOS** — a `.dmg` containing the `.app` and an `/Applications` link. Open it and drag the app across. The bundle is ad-hoc signed and not notarized, so the first launch still needs **Open Anyway**, and the DMG is built on a macOS runner because it needs `hdiutil`.
- **Linux** — an AppImage (`GuildSync-Voice-Mute-<version>-Linux-<arch>.AppImage`, such as `-Linux-x86_64.AppImage`, named for the runner's `uname -m`). An AppImage is used rather than a `.deb` because it carries the GTK4/WebKitGTK 6.0 runtime it links against, so one file runs on Debian, Fedora, Arch and everything else instead of only on the distribution whose package manager built it. `chmod +x` it and run it; it needs no root and installs nothing, and the architecture is part of the name because the bundled runtime is architecture-specific.

Linux installs from v2.0.5 — the last release that shipped a `.deb` — need one manual step. That build looks for a `.deb` when it checks for updates and no later release has one, so it reports the newer version, says the release has no installer for this platform, and offers no download: **Open download page** goes to the release, where the AppImage is fetched by hand. Running the AppImage keeps the saved login and shortcut settings, because both builds read the same user configuration directory, but **Start at login** stores the path of the executable that enabled it, so re-enable it from the AppImage. The packaged copy can then be removed with `sudo apt remove guildsync-voice-mute`.

A tag build verifies each installer on the platform that built it, uploads them to a draft and publishes only once every file is there, so a release cannot ship without its installers. Uploads never replace an existing asset and each published file is listed in `SHA256SUMS`, so a version number always refers to the same bytes. Nothing installs as a service or runs at boot: a login item is only registered when you tick **Start at login**. Uninstalling removes the login item's target with the app, so switching the option off first avoids a stale entry.

## Tray, start at login and updates

A tray icon is added when the desktop exposes one. Closing the window then hides it and the shortcut keeps working; **Quit** in the tray menu exits. Where no tray host is reachable (some Wayland sessions without a StatusNotifier watcher or panel), the app skips the tray and closing the window exits, so the app can never become unreachable. A click on the tray icon shows or hides the window and the tray menu holds **Quit**. The tooltip shows whether a mute is active, and macOS shows a 🔇 status label while muted.

**Start at login** is registered through Wails' autostart manager, which writes the platform's own per-user entry: the `Run` key on Windows, the app registration on macOS, or an XDG autostart `.desktop` file on Linux. Disabling it removes that entry; the desktop's own entry is the source of truth, not a saved setting. Registration takes effect at the next login, not immediately, and launches the app with its window rather than hidden.

The portable zip has no installation step, and the installers put the app in a fixed location. Either way the entry stores the absolute path of the executable that enabled it (symlinks are resolved, so a stable link such as a Homebrew or Scoop shim survives upgrades), so a portable copy must stay where it was extracted. Moving or renaming the app afterwards leaves the entry aimed at a missing file, and because the current state is matched against the running executable's path, the checkbox then reports off until the entry is re-created from the new location.

On macOS 13 and later the bundled `.app` registers through `SMAppService` as a Login Item, so it only starts once it is approved in **System Settings → General → Login Items** (**Login Items & Extensions** on macOS 15 and later), and the bundle should stay in a permanent location such as `/Applications` so that approval keeps pointing at it. macOS pre-13, or a binary run unbundled from `Contents/MacOS`, falls back to a LaunchAgent at `~/Library/LaunchAgents/me.perdues.guildsync.voicemute.plist`. Windows and Linux need no approval step; a stale `Run` value or `.desktop` file is the usual reason an entry survives after the app has moved.

**Updates are checked automatically.** The app queries the GitHub releases API for the newest published release and compares it with the embedded version once at startup, then every 30 minutes from a timer in the Go process. The timer lives outside the window because a window hidden in the tray throttles webview timers, which would let the promised interval slip; a check that cannot reach GitHub is not shown, so being offline or offline-at-launch reports nothing rather than an error.

While the running build is current, the update area is not shown at all: there is no "you are up to date" row and no check button. When a newer published release exists, a row appears below the settings saying `A new version <version> is available.` with a single **Download Update** button. It picks that release's installer for the platform and architecture it is running (`...-Setup.exe`, `...-macOS.dmg`, or the `...-Linux-<arch>.AppImage` built for its architecture), downloads it from the release's own download URL into the user's Downloads folder, verifies the SHA-256 digest GitHub publishes for the asset when there is one, and hands the installer to the desktop so the user runs it. Only an asset whose URL starts with this repository's release download path is accepted, and a checksum mismatch deletes the file. Installation itself is the platform's installer, not a self-replacement: quit the companion, finish the install, then start the new version. A release that carries only the zipped installer gets its zip downloaded, and the row then says to unpack it first. When a release has no installer for the platform, or the download fails, the same button becomes **Open download page** and opens only `github.com/skperdue68/GuildSync-Voice-Mute/releases` URLs in the system browser; the row says which case applied.

## Current verification boundary

Automated tests cover the shortcut parser, mute timer lifecycle, authentication and transport. CI is configured for native Windows, macOS and Linux builds. A live desktop/Discord integration check is still required on each target platform, particularly Wayland portal binding. Do not use a build result as evidence that those checks were performed.
