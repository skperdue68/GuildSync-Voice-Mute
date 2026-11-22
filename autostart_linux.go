//go:build linux

package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func startAtLoginSupported() bool { return true }

func autostartEntryPath() (string, error) {
	root := os.Getenv("XDG_CONFIG_HOME")
	if root == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		root = filepath.Join(home, ".config")
	}
	return filepath.Join(root, "autostart", "guildsync-voice-mute.desktop"), nil
}

func startAtLoginActive() bool {
	path, err := autostartEntryPath()
	if err != nil {
		return false
	}
	_, err = os.Stat(path)
	return err == nil
}

func setStartAtLogin(enabled bool) error {
	path, err := autostartEntryPath()
	if err != nil {
		return err
	}
	if !enabled {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove autostart entry: %w", err)
		}
		return nil
	}
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	entry := fmt.Sprintf(`[Desktop Entry]
Type=Application
Name=GuildSync Voice Mute
Comment=GuildSync voice-channel mute companion
Exec="%s"
Terminal=false
X-GNOME-Autostart-enabled=true
`, executable)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(entry), 0644); err != nil {
		return fmt.Errorf("write autostart entry: %w", err)
	}
	return nil
}
