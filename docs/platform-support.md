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

## Tray, start at login and updates

A tray icon is added when the desktop exposes one. Closing the window then hides it and the shortcut keeps working; **Quit** in the tray menu exits. Where no tray host is reachable (some Wayland sessions without a StatusNotifier watcher or panel), the app skips the tray and closing the window exits, so the app can never become unreachable. A click on the tray icon shows or hides the window and the tray menu holds **Quit**. The tooltip shows whether a mute is active, and macOS shows a 🔇 status label while muted.

**Start at login** is registered through Wails' autostart manager, which writes the platform's own per-user entry: the `Run` key on Windows, the app registration on macOS, or an XDG autostart `.desktop` file on Linux. Disabling it removes that entry; the desktop's own entry is the source of truth, not a saved setting. Registration takes effect at the next login, not immediately, and launches the app with its window rather than hidden.

No installer or system-wide installation is involved. The entry stores the absolute path of the executable that enabled it (symlinks are resolved, so a stable link such as a Homebrew or Scoop shim survives upgrades), so a portable copy must stay where it was extracted. Moving or renaming the app afterwards leaves the entry aimed at a missing file, and because the current state is matched against the running executable's path, the checkbox then reports off until the entry is re-created from the new location.

On macOS 13 and later the bundled `.app` registers through `SMAppService` as a Login Item, so it only starts once it is approved in **System Settings → General → Login Items** (**Login Items & Extensions** on macOS 15 and later), and the bundle should stay in a permanent location such as `/Applications` so that approval keeps pointing at it. macOS pre-13, or a binary run unbundled from `Contents/MacOS`, falls back to a LaunchAgent at `~/Library/LaunchAgents/me.perdues.guildsync.voicemute.plist`. Windows and Linux need no approval step; a stale `Run` value or `.desktop` file is the usual reason an entry survives after the app has moved.

**Check for updates** queries the GitHub releases API for the newest published release and compares it with the embedded version. The app does not download or execute anything: **Open download page** opens only `github.com/skperdue68/GuildSync-Voice-Mute/releases` URLs in the system browser.

## Current verification boundary

Automated tests cover the shortcut parser, mute timer lifecycle, authentication and transport. CI is configured for native Windows, macOS and Linux builds. A live desktop/Discord integration check is still required on each target platform, particularly Wayland portal binding. Do not use a build result as evidence that those checks were performed.
