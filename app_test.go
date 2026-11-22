package main

import (
	"errors"
	"testing"
	"time"
)

type shortcutTestApp struct {
	*App
	registered      map[string]func()
	armed           []time.Duration
	fire            func()
	edges           []bool
	status          []string
	unregisterCalls int
}

func newShortcutTestApp(settings Settings) *shortcutTestApp {
	h := &shortcutTestApp{
		registered: map[string]func(){},
		App: &App{
			settings: settings,
			session:  Session{LoggedIn: true, Allowed: true, Token: "test"},
		},
	}
	h.App.registerShortcut = func(accelerator string, callback func()) error {
		h.registered[accelerator] = callback
		return nil
	}
	h.App.unregisterShortcut = func(accelerator string) error {
		h.unregisterCalls++
		delete(h.registered, accelerator)
		return nil
	}
	h.App.scheduleUnmute = func(delay time.Duration, callback func()) func() {
		h.armed = append(h.armed, delay)
		h.fire = callback
		return func() { h.fire = nil }
	}
	h.App.emitEdge = func(down bool) { h.edges = append(h.edges, down) }
	h.App.emitStatus = func(message string) { h.status = append(h.status, message) }
	return h
}

func TestPressMutesThenUnmutesOnTimeout(t *testing.T) {
	h := newShortcutTestApp(Settings{Enabled: true, Shortcut: "Ctrl+M", TimeoutSeconds: 7})
	if err := h.SetShortcutActive(true); err != nil {
		t.Fatal(err)
	}
	press, ok := h.registered["Ctrl+M"]
	if !ok {
		t.Fatal("shortcut not registered")
	}
	press()
	if !h.muted || len(h.edges) != 1 || !h.edges[0] {
		t.Fatalf("press did not start a mute: muted=%v edges=%v", h.muted, h.edges)
	}
	if len(h.armed) != 1 || h.armed[0] != 7*time.Second {
		t.Fatalf("configured timeout not armed: %v", h.armed)
	}
	fire := h.fire
	fire()
	if h.muted || len(h.edges) != 2 || h.edges[1] {
		t.Fatalf("timeout did not unmute: muted=%v edges=%v", h.muted, h.edges)
	}
	fire()
	if len(h.edges) != 2 {
		t.Fatalf("expired mute released twice: %v", h.edges)
	}
}

func TestPressAgainRestartsTheCountdown(t *testing.T) {
	h := newShortcutTestApp(Settings{Enabled: true, Shortcut: "Ctrl+M", TimeoutSeconds: 5})
	if err := h.SetShortcutActive(true); err != nil {
		t.Fatal(err)
	}
	h.registered["Ctrl+M"]()
	first := h.fire
	h.registered["Ctrl+M"]()
	if len(h.edges) != 1 {
		t.Fatalf("extending re-sent the press edge: %v", h.edges)
	}
	if len(h.armed) != 2 || h.armed[1] != 5*time.Second {
		t.Fatalf("countdown not restarted: %v", h.armed)
	}
	if len(h.status) != 2 || h.status[1] != "Muted — auto-unmute restarted, 5 seconds remaining." {
		t.Fatalf("missing extension status: %v", h.status)
	}
	first()
	if !h.muted {
		t.Fatal("stale timer ended the extended mute")
	}
	h.fire()
	if h.muted {
		t.Fatal("restarted timer did not end the mute")
	}
}

func TestDisablingEndsMuteAndUnregisters(t *testing.T) {
	h := newShortcutTestApp(Settings{Enabled: true, Shortcut: "Alt+M", TimeoutSeconds: 10})
	if err := h.SetShortcutActive(true); err != nil {
		t.Fatal(err)
	}
	h.registered["Alt+M"]()
	if err := h.SetShortcutActive(false); err != nil {
		t.Fatal(err)
	}
	if h.muted {
		t.Fatal("mute survived disabling")
	}
	if len(h.edges) != 2 || h.edges[1] {
		t.Fatalf("release edge missing: %v", h.edges)
	}
	if len(h.registered) != 0 || h.unregisterCalls == 0 {
		t.Fatalf("shortcut still registered: %v", h.registered)
	}
}

func TestRegistrationFailureIsRememberedForTheSameShortcutOnly(t *testing.T) {
	calls := 0
	a := &App{settings: Settings{Enabled: true, Shortcut: "Ctrl+M", TimeoutSeconds: 10}, session: Session{LoggedIn: true, Allowed: true, Token: "test"}}
	a.registerShortcut = func(string, func()) error {
		calls++
		return errors.New("the shortcut is already in use")
	}
	for i := 0; i < 3; i++ {
		if a.SetShortcutActive(true) == nil {
			t.Fatal("failed registration activated the shortcut")
		}
	}
	if calls != 1 {
		t.Fatalf("retried the failing registration %d times", calls)
	}
	state := a.GetShortcutAvailability()
	if state.Available || state.Reason == "" {
		t.Fatal("missing availability reason")
	}
	a.shortcutMu.Lock()
	a.settings.Shortcut = "Ctrl+Shift+M"
	a.shortcutMu.Unlock()
	if err := a.SetShortcutActive(true); err == nil {
		t.Fatal("second failure reported success")
	}
	if calls != 2 {
		t.Fatalf("distinct shortcut not retried: %d calls", calls)
	}
	if a.GetShortcutSettings().Shortcut != "Ctrl+Shift+M" {
		t.Fatal("settings inaccessible")
	}
}
