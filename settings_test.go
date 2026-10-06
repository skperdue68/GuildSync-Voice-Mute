package main

import (
	"path/filepath"
	"testing"
)

func TestShortcutValidation(t *testing.T) {
	for _, s := range []string{"", "Ctrl+Ctrl+M", "Ctrl+Nope", "Alt+F4"} {
		if _, err := parseShortcut(s); err == nil {
			t.Errorf("accepted %q", s)
		}
	}
	for _, input := range []string{"M", "F8", "Ctrl", "Space", "M+N", "Ctrl+M+N", "Ctrl+Shift+M+N"} {
		if _, err := parseShortcut(input); err != nil {
			t.Errorf("rejected %s: %v", input, err)
		}
	}
	s, err := parseShortcut("shift+ctrl+m")
	if err != nil || s.Label != "Ctrl+Shift+M" {
		t.Fatalf("%+v %v", s, err)
	}
}
func TestSettingsRoundTripAndCorruption(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	defaults := readSettings(path)
	if defaults.Enabled || defaults.Shortcut != "Ctrl+M" {
		t.Fatal(defaults)
	}
	if err := writeJSON(path, Settings{Enabled: true, Shortcut: "Alt+F2"}); err != nil {
		t.Fatal(err)
	}
	if got := readSettings(path); !got.Enabled || got.Shortcut != "Alt+F2" {
		t.Fatal(got)
	}
	if err := writeJSON(path, Settings{Enabled: true, Shortcut: "garbage"}); err != nil {
		t.Fatal(err)
	}
	if readSettings(path).Enabled {
		t.Fatal("invalid setting activated")
	}
}
func TestHeldStateRequiresRelease(t *testing.T) {
	s := holdState{blocked: true}
	if s.update(true, true) != "" {
		t.Fatal("startup replay")
	}
	s.update(false, true)
	if s.update(true, true) != "pressed" {
		t.Fatal("no press")
	}
	if s.update(true, false) != "released" {
		t.Fatal("no release on disable")
	}
	if s.update(true, true) != "" {
		t.Fatal("reconnect replay")
	}
	s.update(false, true)
	if s.update(true, true) != "pressed" {
		t.Fatal("not rearmed")
	}
}
