package main

import (
	"path/filepath"
	"testing"
)

func TestShortcutValidation(t *testing.T) {
	for _, s := range []string{"", "Ctrl", "Ctrl+Ctrl+M", "Ctrl+Nope", "Alt+F4", "M+N", "Ctrl+M+N", "Ctrl+Shift+M+N"} {
		if _, err := parseShortcut(s); err == nil {
			t.Errorf("accepted %q", s)
		}
	}
	for _, input := range []string{"M", "F8", "Space", "Ctrl+M", "Ctrl+Shift+M", "Alt+Shift+F2", "Ctrl+PageUp"} {
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
	if defaults.Enabled || defaults.Shortcut != "Ctrl+M" || defaults.TimeoutSeconds != defaultTimeoutSeconds {
		t.Fatal(defaults)
	}
	if err := writeJSON(path, Settings{Enabled: true, Shortcut: "Alt+F2", TimeoutSeconds: 25}); err != nil {
		t.Fatal(err)
	}
	if got := readSettings(path); !got.Enabled || got.Shortcut != "Alt+F2" || got.TimeoutSeconds != 25 {
		t.Fatal(got)
	}
	if err := writeJSON(path, Settings{Enabled: true, Shortcut: "M+N", TimeoutSeconds: 25}); err != nil {
		t.Fatal(err)
	}
	if readSettings(path).Enabled {
		t.Fatal("invalid setting activated")
	}
}
func TestTimeoutIsClamped(t *testing.T) {
	for _, tc := range []struct{ in, want int }{{0, 10}, {-4, 10}, {1, 3}, {3, 3}, {60, 60}, {120, 120}, {900, 120}} {
		if got := clampTimeoutSeconds(tc.in); got != tc.want {
			t.Errorf("clampTimeoutSeconds(%d) = %d, want %d", tc.in, got, tc.want)
		}
	}
}
