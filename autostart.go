package main

import "fmt"

// Start-at-login is registered with the desktop itself, so the operating system
// entry is the source of truth rather than a settings field that can drift from
// it. The platform files implement the three helpers below.
func (a *App) StartAtLoginSupported() bool { return startAtLoginSupported() }

func (a *App) GetStartAtLogin() bool { return startAtLoginActive() }

func (a *App) SetStartAtLogin(enabled bool) error {
	if !startAtLoginSupported() {
		return fmt.Errorf("starting at login is not supported on this platform")
	}
	return setStartAtLogin(enabled)
}
