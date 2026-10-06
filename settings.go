package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Settings struct {
	Enabled  bool   `json:"enabled"`
	Shortcut string `json:"shortcut"`
}
type Shortcut struct {
	Keys  []int
	Label string
}

func parseShortcut(input string) (Shortcut, error) {
	result := Shortcut{}
	parts := strings.Split(strings.ToUpper(strings.TrimSpace(input)), "+")
	seen := map[string]bool{}
	mods := map[string]int{"CTRL": 17, "ALT": 18, "SHIFT": 16}
	for _, part := range parts[:len(parts)-1] {
		part = strings.TrimSpace(part)
		key, ok := mods[part]
		if !ok || seen[part] {
			return result, fmt.Errorf("use Ctrl, Alt or Shift plus a letter, number or F1–F12")
		}
		seen[part] = true
		_ = key
	}
	key := strings.TrimSpace(parts[len(parts)-1])
	code := 0
	if len(key) == 1 && ((key[0] >= 'A' && key[0] <= 'Z') || (key[0] >= '0' && key[0] <= '9')) {
		code = int(key[0])
	}
	for n := 1; n <= 12; n++ {
		if key == fmt.Sprintf("F%d", n) {
			code = 111 + n
		}
	}
	if len(seen) == 0 || code == 0 || ((seen["ALT"] || seen["CTRL"]) && key == "F4") {
		return result, fmt.Errorf("use Ctrl, Alt or Shift plus a letter, number or F1–F12; Alt+F4 and Ctrl+F4 are reserved")
	}
	labels := []string{}
	for _, mod := range []string{"CTRL", "ALT", "SHIFT"} {
		if seen[mod] {
			result.Keys = append(result.Keys, mods[mod])
			labels = append(labels, map[string]string{"CTRL": "Ctrl", "ALT": "Alt", "SHIFT": "Shift"}[mod])
		}
	}
	result.Keys = append(result.Keys, code)
	labels = append(labels, key)
	result.Label = strings.Join(labels, "+")
	return result, nil
}
func dataPath(name string) (string, error) {
	root, err := os.UserConfigDir()
	return filepath.Join(root, "GuildSync-Voice-Mute", name), err
}
func readSettings(path string) Settings {
	fallback := Settings{Shortcut: "Ctrl+M"}
	var s Settings
	data, err := os.ReadFile(path)
	if err != nil || json.Unmarshal(data, &s) != nil {
		return fallback
	}
	parsed, err := parseShortcut(s.Shortcut)
	if err != nil {
		return fallback
	}
	s.Shortcut = parsed.Label
	return s
}
func writeJSON(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".guildsync-*")
	if err != nil {
		return err
	}
	name := f.Name()
	defer os.Remove(name)
	if err = f.Chmod(0600); err == nil {
		_, err = f.Write(data)
	}
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Rename(name, path)
}

type holdState struct{ held, blocked bool }

func (s *holdState) update(down, allowed bool) string {
	if !down {
		s.blocked = false
		if s.held {
			s.held = false
			return "released"
		}
		return ""
	}
	if !allowed {
		s.blocked = true
		if s.held {
			s.held = false
			return "released"
		}
		return ""
	}
	if !s.held && !s.blocked {
		s.held = true
		return "pressed"
	}
	return ""
}
