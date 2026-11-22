//go:build linux

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAutostartEntryLifecycle(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", root)
	if startAtLoginActive() {
		t.Fatal("autostart active before it was written")
	}
	if err := setStartAtLogin(true); err != nil {
		t.Fatal(err)
	}
	if !startAtLoginActive() || !(&App{}).GetStartAtLogin() {
		t.Fatal("autostart entry missing")
	}
	path := filepath.Join(root, "autostart", "guildsync-voice-mute.desktop")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "Exec=\"") || !strings.Contains(string(data), "GuildSync Voice Mute") {
		t.Fatalf("unexpected entry:\n%s", data)
	}
	if err := setStartAtLogin(false); err != nil {
		t.Fatal(err)
	}
	if startAtLoginActive() {
		t.Fatal("autostart entry survived removal")
	}
}
