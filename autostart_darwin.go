//go:build darwin

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const launchAgentLabel = "me.perdues.guildsync.voicemute"

func startAtLoginSupported() bool { return true }

func launchAgentPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Library", "LaunchAgents", launchAgentLabel+".plist"), nil
}

// loginItemTarget prefers the application bundle so macOS launches the app the
// way Finder does, and falls back to the executable for unbundled builds.
func loginItemTarget() (string, error) {
	executable, err := os.Executable()
	if err != nil {
		return "", err
	}
	if index := strings.Index(executable, "/Contents/MacOS/"); index > 0 {
		return executable[:index], nil
	}
	return executable, nil
}

func startAtLoginActive() bool {
	path, err := launchAgentPath()
	if err != nil {
		return false
	}
	_, err = os.Stat(path)
	return err == nil
}

func setStartAtLogin(enabled bool) error {
	path, err := launchAgentPath()
	if err != nil {
		return err
	}
	if !enabled {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove login item: %w", err)
		}
		return nil
	}
	target, err := loginItemTarget()
	if err != nil {
		return err
	}
	plist := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>%s</string>
	<key>ProgramArguments</key>
	<array>
		<string>%s</string>
	</array>
	<key>RunAtLoad</key>
	<true/>
	<key>ProcessType</key>
	<string>Interactive</string>
</dict>
</plist>
`, launchAgentLabel, target)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(plist), 0644); err != nil {
		return fmt.Errorf("write login item: %w", err)
	}
	return nil
}
