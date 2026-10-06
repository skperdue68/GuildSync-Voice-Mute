package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
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
	codes := map[string]int{"CTRL": 17, "ALT": 18, "SHIFT": 16, "SPACE": 32, "TAB": 9, "ENTER": 13, "BACKSPACE": 8, "DELETE": 46, "INSERT": 45, "HOME": 36, "END": 35, "PAGEUP": 33, "PAGEDOWN": 34, "LEFT": 37, "UP": 38, "RIGHT": 39, "DOWN": 40}
	labels := map[string]string{"CTRL": "Ctrl", "ALT": "Alt", "SHIFT": "Shift", "SPACE": "Space", "TAB": "Tab", "ENTER": "Enter", "BACKSPACE": "Backspace", "DELETE": "Delete", "INSERT": "Insert", "HOME": "Home", "END": "End", "PAGEUP": "PageUp", "PAGEDOWN": "PageDown", "LEFT": "Left", "UP": "Up", "RIGHT": "Right", "DOWN": "Down"}
	seen := map[string]bool{}
	for _, part := range strings.Split(strings.ToUpper(strings.TrimSpace(input)), "+") {
		key := strings.TrimSpace(part)
		if seen[key] {
			return result, fmt.Errorf("duplicate shortcut key")
		}
		seen[key] = true
		code := codes[key]
		if len(key) == 1 && ((key[0] >= 'A' && key[0] <= 'Z') || (key[0] >= '0' && key[0] <= '9')) {
			code = int(key[0])
		}
		for n := 1; n <= 12; n++ {
			if key == fmt.Sprintf("F%d", n) {
				code = 111 + n
			}
		}
		if code == 0 {
			return result, fmt.Errorf("use one or more letters, numbers, F1–F12, Ctrl, Alt, Shift, or supported navigation keys")
		}
		result.Keys = append(result.Keys, code)
	}
	if ((seen["ALT"] || seen["CTRL"]) && seen["F4"]) || (seen["ALT"] && seen["TAB"]) || (seen["CTRL"] && seen["ALT"] && seen["DELETE"]) {
		return result, fmt.Errorf("reserved system shortcut")
	}
	order := func(code int) int {
		switch code {
		case 17:
			return -3
		case 18:
			return -2
		case 16:
			return -1
		}
		return code
	}
	sort.Slice(result.Keys, func(i, j int) bool { return order(result.Keys[i]) < order(result.Keys[j]) })
	canonical := []string{}
	for _, code := range result.Keys {
		label := ""
		for key, value := range codes {
			if value == code {
				label = labels[key]
			}
		}
		if label == "" {
			if code >= 112 {
				label = fmt.Sprintf("F%d", code-111)
			} else {
				label = string(rune(code))
			}
		}
		canonical = append(canonical, label)
	}
	result.Label = strings.Join(canonical, "+")
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
