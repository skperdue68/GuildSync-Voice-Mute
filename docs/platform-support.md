# Platform support and permissions

## Windows

Read the global keyboard state with `GetAsyncKeyState` every 25 milliseconds. No keys are intercepted or suppressed. The initial held state is blocked until all required keys have been released. WebView2 is needed for the GUI. Secure desktops, lock screens and disconnected sessions do not provide an ordinary interactive keyboard context.

## macOS

Use CoreGraphics global key-state observation. Allow the application in **System Settings → Privacy & Security → Input Monitoring**, then restart and enable it again. If permission is absent, activation fails with instructions. macOS can reserve shortcuts or treat function keys as media controls; use an ordinary modifier/letter combination when necessary. The app is not signed or notarized by this project. The current macOS letter key-code mapping assumes a QWERTY layout; non-QWERTY layouts need manual verification or a function/navigation-key shortcut. Windows and X11 letter capture follows the active layout.

The permission check uses Apple's [CGPreflightListenEventAccess](https://developer.apple.com/documentation/coregraphics/cgpreflightlisteneventaccess()). Native key-state observation runs outside network operations. Manual testing on macOS is required before declaring the global hold feature verified.

Input Monitoring is an operating-system permission. Installation cannot grant it and this application cannot bypass it. Grant permission yourself, restart the app, and enable the shortcut.

## Linux X11

Requires an accessible X11 display and `libX11`. The listener queries key state instead of grabbing or consuming the keys. Shortcut key names are resolved against the keyboard layout when activation begins; restart/re-enable after changing your keyboard layout.

## Linux Wayland

Requires a desktop implementing the [GlobalShortcuts portal](https://flatpak.github.io/xdg-desktop-portal/docs/doc-org.freedesktop.portal.GlobalShortcuts.html). The desktop may show its own approval/binding dialog. Its actual binding is authoritative and may differ from the app's requested shortcut; check the dialog. Declining permission or a missing backend leaves activation unavailable. The app never falls back to a focused-window shortcut.

The portal supports one ordinary key (such as F8 or M), optionally with Ctrl/Alt/Shift. Modifier-only shortcuts and multiple ordinary keys (such as M+N) cannot be activated through this API and are rejected with an explanation; these combinations work on Windows, macOS, and X11.

Activation and deactivation signals drive hold/release. Portal support varies by desktop; install the appropriate `xdg-desktop-portal` backend. A portal timeout or disconnect ends local activity and the bot's session expiry remains the safety fallback.

A missing GlobalShortcuts API or inaccessible desktop session does not fail installation or application startup. The app shows **Mute is unavailable for this Linux version/session** and disables Enable after detecting the missing API. It remembers that result for the current app session; restart after changing desktop support. A declined or timed-out consent request is not automatically retried on socket reconnect. To try consent again, explicitly change/re-enable the shortcut. The app does not request broader permissions to compensate for a missing API.

## Current verification boundary

Automated tests cover parser, authentication and transport lifecycle. CI is configured for native Windows, macOS and Linux builds. A live desktop/Discord integration check is still required on each target platform, particularly Wayland binding and macOS permission behavior. Do not use a build result as evidence that those checks were performed.

