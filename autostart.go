package main

import (
	"errors"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// Wails registers the login entry with the platform itself, so the operating
// system entry is the source of truth instead of a settings field that can drift
// from it. The identifier keeps the entry named after the app on every platform
// and matches the single-instance ID.
const autostartIdentifier = "me.perdues.guildsync.voicemute"

func (a *App) StartAtLoginSupported() bool {
	return a.app != nil && a.app.Autostart != nil
}

func (a *App) GetStartAtLogin() bool {
	if !a.StartAtLoginSupported() {
		return false
	}
	enabled, err := a.app.Autostart.IsEnabled()
	return err == nil && enabled
}

func (a *App) SetStartAtLogin(enabled bool) error {
	if !a.StartAtLoginSupported() {
		return errors.New("starting at login is unavailable")
	}
	if !enabled {
		return a.app.Autostart.Disable()
	}
	return a.app.Autostart.EnableWithOptions(application.AutostartOptions{Identifier: autostartIdentifier})
}
