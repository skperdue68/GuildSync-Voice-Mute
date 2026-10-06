package main

import (
	"context"
	"fmt"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"net/http"
	"sync"
)

type App struct {
	mu                 sync.Mutex
	ctx                context.Context
	loginState         string
	oauthServer        *http.Server
	session            Session
	authGeneration     uint64
	sessionPath        string
	verifySession      func(Session) error
	shortcutMu         sync.Mutex
	settings           Settings
	stopListener       func()
	hold               holdState
	active             bool
	listenerGeneration uint64
	emitEdge           func(bool)
}

func NewApp() *App {
	path, _ := dataPath("settings.json")
	return &App{settings: readSettings(path), hold: holdState{blocked: true}}
}
func (a *App) startup(ctx context.Context)  { a.ctx = ctx }
func (a *App) shutdown(ctx context.Context) { _ = a.SetShortcutActive(false); a.stopOAuth() }
func (a *App) emit(event string, value any) {
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, event, value)
	}
}
func (a *App) GetShortcutSettings() Settings {
	a.shortcutMu.Lock()
	defer a.shortcutMu.Unlock()
	return a.settings
}
func (a *App) SetShortcutSettings(s Settings) error {
	parsed, err := parseShortcut(s.Shortcut)
	if err != nil {
		return err
	}
	s.Shortcut = parsed.Label
	if err = a.SetShortcutActive(false); err != nil {
		return err
	}
	path, err := dataPath("settings.json")
	if err != nil {
		return err
	}
	if err = writeJSON(path, s); err != nil {
		return err
	}
	a.shortcutMu.Lock()
	a.settings = s
	a.shortcutMu.Unlock()
	return nil
}
func (a *App) SetShortcutActive(active bool) error {
	a.shortcutMu.Lock()
	a.listenerGeneration++
	generation := a.listenerGeneration
	if a.stopListener != nil {
		a.stopListener()
		a.stopListener = nil
	}
	a.active = false
	if state := a.hold.update(true, false); state != "" {
		a.emit("voice-shortcut-edge", false)
	}
	if !active {
		a.shortcutMu.Unlock()
		return nil
	}
	settings := a.settings
	a.shortcutMu.Unlock()
	if !settings.Enabled {
		return nil
	}
	a.mu.Lock()
	s := a.session
	a.mu.Unlock()
	if !s.LoggedIn || !s.Allowed || (s.User.Role != "user" && s.User.Role != "admin") {
		return fmt.Errorf("Approved User or Admin access is required.")
	}
	shortcut, err := parseShortcut(settings.Shortcut)
	if err != nil {
		return err
	}
	stop, err := startShortcut(shortcut, func(down bool) { a.observeEdge(generation, down) }, func(err error) {
		a.shortcutMu.Lock()
		defer a.shortcutMu.Unlock()
		if generation != a.listenerGeneration {
			return
		}
		a.active = false
		a.hold.update(true, false)
		a.emit("shortcut-error", err.Error())
	})
	if err != nil {
		return err
	}
	a.shortcutMu.Lock()
	if generation != a.listenerGeneration {
		a.shortcutMu.Unlock()
		stop()
		return nil
	}
	a.stopListener = stop
	a.active = true
	a.shortcutMu.Unlock()
	return nil
}

func (a *App) observeEdge(generation uint64, down bool) {
	a.shortcutMu.Lock()
	defer a.shortcutMu.Unlock()
	if generation != a.listenerGeneration {
		return
	}
	wasBlocked := a.hold.blocked
	state := a.hold.update(down, a.active)
	if state != "" || (!down && wasBlocked) {
		if a.emitEdge != nil {
			a.emitEdge(state == "pressed")
		} else {
			a.emit("voice-shortcut-edge", state == "pressed")
		}
	}
}
