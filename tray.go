package main

import (
	_ "embed"
	"runtime"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/icons"
)

//go:embed build/appicon.png
var appIcon []byte

// startTray adds a tray icon when the desktop actually shows one. Closing the
// window then hides it, which is what keeps the shortcut registered while the
// user is in Discord. A desktop without a tray host keeps close-to-quit, because
// there would be no way back to the window.
func (a *App) startTray() {
	window := a.currentWindow()
	if a.app == nil || window == nil || !trayHostAvailable() {
		return
	}
	tray := a.app.SystemTray.New()
	if runtime.GOOS == "darwin" {
		tray.SetTemplateIcon(icons.SystrayMacTemplate)
	} else {
		tray.SetIcon(appIcon)
	}
	// Attaching the window brings the platform defaults: a click toggles the
	// window and a right click opens the menu.
	tray.AttachWindow(window)
	menu := a.app.NewMenu()
	menu.Add("Quit GuildSync Voice Mute").OnClick(func(*application.Context) { a.Quit() })
	tray.SetMenu(menu)
	a.mu.Lock()
	a.tray = tray
	a.mu.Unlock()
	a.updateTrayIndicator(false)
}

// updateTrayIndicator mirrors the mute state in the tray so the window can stay
// hidden. The tooltip carries it on every platform; the menu bar label is the
// only place macOS shows text next to the icon.
func (a *App) updateTrayIndicator(muted bool) {
	a.mu.Lock()
	tray := a.tray
	a.mu.Unlock()
	if tray == nil {
		return
	}
	tooltip := "GuildSync Voice Mute"
	label := ""
	if muted {
		tooltip += " — muted"
		if runtime.GOOS == "darwin" {
			label = "🔇"
		}
	}
	tray.SetTooltip(tooltip)
	tray.SetLabel(label)
}

func (a *App) currentWindow() *application.WebviewWindow {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.window
}

func (a *App) setWindow(window *application.WebviewWindow) {
	a.mu.Lock()
	a.window = window
	a.mu.Unlock()
}

// ShowWindow brings the window back, which is what a second launch of the app
// asks for instead of starting its own copy.
func (a *App) ShowWindow() {
	if window := a.currentWindow(); window != nil {
		window.Show()
		window.Focus()
	}
}

// hideOnClose reports whether closing the window should hide it instead of
// quitting. Without a tray icon the window is the only way back into the app.
func (a *App) hideOnClose() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.tray != nil && !a.quitting
}

// Quit leaves through the tray menu without the close-to-tray hook canceling the
// shutdown.
func (a *App) Quit() {
	a.mu.Lock()
	a.quitting = true
	app := a.app
	a.mu.Unlock()
	if app != nil {
		app.Quit()
	}
}
