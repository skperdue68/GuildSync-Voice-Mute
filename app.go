package main

import (
	"net/http"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

type App struct {
	mu             sync.Mutex
	app            *application.App
	window         *application.WebviewWindow
	tray           *application.SystemTray
	quitting       bool
	updateURL      string
	loginState     string
	oauthServer    *http.Server
	session        Session
	authGeneration uint64
	sessionPath    string
	verifySession  func(Session) error

	shortcutMu          sync.Mutex
	settings            Settings
	active              bool
	muted               bool
	registered          string
	stopTimer           func()
	timerGeneration     uint64
	shortcutUnavailable error
	unavailableShortcut string
	registerShortcut    func(string, func()) error
	unregisterShortcut  func(string) error
	scheduleUnmute      func(time.Duration, func()) func()
	emitEdge            func(bool)
	emitStatus          func(string)

	// assetBase and openPath let tests stand in for GitHub and the desktop's
	// installer handler; both fall back to the real ones when empty or nil.
	assetBase string
	openPath  func(string) error

	// updateStop ends the background update timer; emitUpdate stands in for the
	// event the window listens to when a check runs.
	updateStop chan struct{}
	emitUpdate func(UpdateInfo)
}

func NewApp() *App {
	path, _ := dataPath("settings.json")
	return &App{settings: readSettings(path)}
}
func (a *App) ServiceShutdown() error {
	_ = a.SetShortcutActive(false)
	a.stopUpdateWatcher()
	a.stopOAuth()
	return nil
}
func (a *App) emit(event string, value any) {
	if a.app != nil {
		a.app.Event.Emit(event, value)
	}
}
func (a *App) GetShortcutSettings() Settings {
	a.shortcutMu.Lock()
	defer a.shortcutMu.Unlock()
	return a.settings
}

// SetShortcutSettings validates and persists the shortcut configuration. It
// stops the running shortcut first, so the caller must reactivate it (the
// frontend does) for a new key or duration to take effect immediately.
func (a *App) SetShortcutSettings(s Settings) error {
	parsed, err := parseShortcut(s.Shortcut)
	if err != nil {
		return err
	}
	s.Shortcut = parsed.Label
	s.TimeoutSeconds = clampTimeoutSeconds(s.TimeoutSeconds)
	if err := a.SetShortcutActive(false); err != nil {
		return err
	}
	path, err := dataPath("settings.json")
	if err != nil {
		return err
	}
	if err := writeJSON(path, s); err != nil {
		return err
	}
	a.shortcutMu.Lock()
	a.settings = s
	a.shortcutMu.Unlock()
	return nil
}
