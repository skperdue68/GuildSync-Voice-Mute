//go:build windows

package main

import (
	"fmt"
	"os"

	"golang.org/x/sys/windows/registry"
)

const (
	autostartKeyPath   = `Software\Microsoft\Windows\CurrentVersion\Run`
	autostartValueName = "GuildSync Voice Mute"
)

func startAtLoginSupported() bool { return true }

func startAtLoginActive() bool {
	key, err := registry.OpenKey(registry.CURRENT_USER, autostartKeyPath, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer key.Close()
	value, _, err := key.GetStringValue(autostartValueName)
	return err == nil && value != ""
}

func setStartAtLogin(enabled bool) error {
	key, _, err := registry.CreateKey(registry.CURRENT_USER, autostartKeyPath, registry.SET_VALUE|registry.QUERY_VALUE)
	if err != nil {
		return fmt.Errorf("open login items: %w", err)
	}
	defer key.Close()
	if !enabled {
		if err := key.DeleteValue(autostartValueName); err != nil && err != registry.ErrNotExist {
			return fmt.Errorf("remove login item: %w", err)
		}
		return nil
	}
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	if err := key.SetStringValue(autostartValueName, fmt.Sprintf("%q", executable)); err != nil {
		return fmt.Errorf("register login item: %w", err)
	}
	return nil
}
