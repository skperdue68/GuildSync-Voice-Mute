package main

import (
	_ "embed"
	"runtime"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/icons"
)

//go:embed build/appicon.png
var appIcon []byte

// startTray adds a tray icon when the desktop has a tray host. The window then
// closes to the tray instead of exiting, which is what keeps the shortcut
// registered while the user is in Discord. Desktops without a tray host keep the
// old behaviour: closing the window quits the app.
func (a *App) startTray() {
	if a.app == nil || !trayHostAvailable() {
		return
	}
	tray := a.app.SystemTray.New()
	if runtime.GOOS == "darwin" {
		tray.SetTemplateIcon(icons.SystrayMacTemplate)
	} else {
		tray.SetIcon(appIcon)
	}
	tray.SetTooltip("GuildSync Voice Mute — idle")
	if runtime.GOOS == "darwin" {
		// A macOS status item opens its menu on a normal click.
		tray.OnClick(tray.ShowMenu)
	} else {
		tray.OnClick(a.ShowWindow)
	}
	tray.OnDoubleClick(a.ShowWindow)
	menu := a.app.NewMenu()
	menu.Add("Show window").OnClick(func(*application.Context) { a.ShowWindow() })
	menu.Add("Hide window").OnClick(func(*application.Context) { a.HideWindow() })
	status := menu.Add("Mute idle")
	status.SetEnabled(false)
	menu.AddSeparator()
	menu.Add("Quit GuildSync Voice Mute").OnClick(func(*application.Context) { a.Quit() })
	tray.SetMenu(menu)
	a.mu.Lock()
	a.tray = tray
	a.trayStatus = status
	a.mu.Unlock()
	a.updateTrayIndicator(false, "")
}

// updateTrayIndicator reflects mute state in the tray so the window can stay
// hidden while the user is in Discord.
func (a *App) updateTrayIndicator(muted bool, message string) {
	a.mu.Lock()
	tray, status := a.tray, a.trayStatus
	a.mu.Unlock()
	if tray == nil {
		return
	}
	label := "Mute idle"
	if muted {
		label = "Muted"
	}
	if message != "" {
		label = message
	}
	status.SetLabel(label)
	tooltip := "GuildSync Voice Mute — idle"
	if muted {
		tooltip = "GuildSync Voice Mute — muted"
	}
	tray.SetTooltip(tooltip)
	if runtime.GOOS == "darwin" {
		if muted {
			tray.SetLabel("🔇")
		} else {
			tray.SetLabel("")
		}
	}
}

func (a *App) stopTray() {
	a.mu.Lock()
	tray := a.tray
	a.tray = nil
	a.trayStatus = nil
	a.mu.Unlock()
	if tray != nil {
		tray.Destroy()
	}
}

func (a *App) setWindow(window *application.WebviewWindow) {
	a.mu.Lock()
	a.window = window
	a.mu.Unlock()
}

func (a *App) ShowWindow() {
	a.mu.Lock()
	w := a.window
	a.mu.Unlock()
	if w == nil {
		return
	}
	w.Show()
	w.Focus()
}

func (a *App) HideWindow() {
	a.mu.Lock()
	w := a.window
	a.mu.Unlock()
	if w != nil {
		w.Hide()
	}
}

// hideOnClose reports whether closing the window should hide it instead of
// quitting. Without a tray icon the window is the only way back into the app.
func (a *App) hideOnClose() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.tray != nil && !a.quitting
}

// Quit leaves through the tray menu or the title bar without the close-to-tray
// hook canceling the shutdown.
func (a *App) Quit() {
	a.mu.Lock()
	a.quitting = true
	app := a.app
	a.mu.Unlock()
	if app != nil {
		app.Quit()
	}
}
