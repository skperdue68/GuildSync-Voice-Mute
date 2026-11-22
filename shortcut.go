package main

import (
	"fmt"
	"time"
)

// The shortcut is press-to-mute, the way the GuildSync client does it: one press
// starts a mute for the configured duration, pressing again restarts the
// countdown, and the mute always ends on its own. Nothing depends on seeing a
// key release, so no platform keyboard listener or input-monitoring permission is
// needed; Wails registers the accelerator through the OS instead.

// registerShortcutFunc registers a system-wide accelerator and calls callback on
// every press.
func (a *App) registerShortcutFunc() func(string, func()) error {
	if a.registerShortcut != nil {
		return a.registerShortcut
	}
	return func(accelerator string, callback func()) error {
		if a.app == nil {
			return &shortcutUnavailableError{"Global shortcuts are unavailable: the application is not running."}
		}
		return a.app.GlobalShortcut.Register(accelerator, callback)
	}
}

func (a *App) unregisterShortcutFunc() func(string) error {
	if a.unregisterShortcut != nil {
		return a.unregisterShortcut
	}
	return func(accelerator string) error {
		if a.app == nil {
			return nil
		}
		return a.app.GlobalShortcut.Unregister(accelerator)
	}
}

// scheduleUnmuteFunc returns a stop function for the automatic unmute.
func (a *App) scheduleUnmuteFunc() func(time.Duration, func()) func() {
	if a.scheduleUnmute != nil {
		return a.scheduleUnmute
	}
	return func(delay time.Duration, callback func()) func() {
		timer := time.AfterFunc(delay, callback)
		return func() { timer.Stop() }
	}
}

func timeoutDuration(seconds int) time.Duration {
	return time.Duration(clampTimeoutSeconds(seconds)) * time.Second
}

// SetShortcutActive arms or disarms the shortcut. Turning it off always ends an
// active mute so the server is told to unmute immediately.
func (a *App) SetShortcutActive(active bool) error {
	if !active {
		a.shortcutMu.Lock()
		a.active = false
		a.releaseMuteLocked()
		a.unregisterLocked()
		a.shortcutMu.Unlock()
		return nil
	}
	a.mu.Lock()
	s := a.session
	a.mu.Unlock()
	if !s.LoggedIn || !s.Allowed || s.Token == "" {
		return fmt.Errorf("Sign in with Discord to enable the shortcut.")
	}
	a.shortcutMu.Lock()
	defer a.shortcutMu.Unlock()
	settings := a.settings
	a.active = true
	if !settings.Enabled {
		return nil
	}
	// A registration failure is remembered for the shortcut that caused it, so a
	// failing portal prompt is not retried on every screen, but assigning a
	// different shortcut tries again.
	if a.shortcutUnavailable != nil && a.unavailableShortcut == settings.Shortcut {
		return a.shortcutUnavailable
	}
	accelerator := settings.Shortcut
	a.unregisterLocked()
	if err := a.registerShortcutFunc()(accelerator, a.pressShortcut); err != nil {
		a.active = false
		unavailable := &shortcutUnavailableError{err.Error()}
		a.shortcutUnavailable = unavailable
		a.unavailableShortcut = accelerator
		return unavailable
	}
	a.registered = accelerator
	return nil
}

func (a *App) unregisterLocked() {
	if a.registered == "" {
		return
	}
	_ = a.unregisterShortcutFunc()(a.registered)
	a.registered = ""
}

// pressShortcut starts a mute, or restarts its countdown when one is already
// running. It never ends a mute; the timer does that.
func (a *App) pressShortcut() {
	a.shortcutMu.Lock()
	defer a.shortcutMu.Unlock()
	if !a.active || !a.settings.Enabled {
		return
	}
	timeout := timeoutDuration(a.settings.TimeoutSeconds)
	if a.muted {
		a.armMuteTimerLocked(timeout)
		a.emitStatusMessage(fmt.Sprintf("Muted — auto-unmute restarted, %d seconds remaining.", int(timeout.Seconds())))
		return
	}
	a.muted = true
	a.emitEdgeEvent(true)
	a.emitStatusMessage(fmt.Sprintf("Muted — auto-unmute in %d seconds.", int(timeout.Seconds())))
	a.armMuteTimerLocked(timeout)
}

// armMuteTimerLocked (re)arms the automatic unmute. Callers must hold shortcutMu.
// Each arming gets its own generation so a timer that already fired cannot end a
// mute that a later press extended.
func (a *App) armMuteTimerLocked(timeout time.Duration) {
	if a.stopTimer != nil {
		a.stopTimer()
	}
	a.timerGeneration++
	generation := a.timerGeneration
	a.stopTimer = a.scheduleUnmuteFunc()(timeout, func() { a.expireMute(generation) })
}

// expireMute ends a mute that has been active for the configured duration.
func (a *App) expireMute(generation uint64) {
	a.shortcutMu.Lock()
	defer a.shortcutMu.Unlock()
	if generation != a.timerGeneration || !a.muted {
		return
	}
	a.muted = false
	a.stopTimer = nil
	a.emitEdgeEvent(false)
	a.emitStatusMessage("Voice mute ended automatically.")
}

// releaseMuteLocked clears an active mute and reports the release edge once.
// Callers must hold shortcutMu.
func (a *App) releaseMuteLocked() {
	if a.stopTimer != nil {
		a.stopTimer()
		a.stopTimer = nil
	}
	if !a.muted {
		return
	}
	a.muted = false
	a.emitEdgeEvent(false)
}

func (a *App) emitEdgeEvent(down bool) {
	a.updateTrayIndicator(down)
	if a.emitEdge != nil {
		a.emitEdge(down)
		return
	}
	a.emit(eventVoiceShortcutEdge, down)
}

func (a *App) emitStatusMessage(message string) {
	if a.emitStatus != nil {
		a.emitStatus(message)
		return
	}
	a.emit(eventVoiceShortcutStatus, message)
}
