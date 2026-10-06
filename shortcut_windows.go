//go:build windows

package main

import (
	"fmt"
	"syscall"
)

func startShortcut(shortcut Shortcut, edge func(bool), failed func(error)) (func(), error) {
	user32 := syscall.NewLazyDLL("user32.dll")
	keyState := user32.NewProc("GetAsyncKeyState")
	if err := keyState.Find(); err != nil {
		return nil, fmt.Errorf("Global keyboard access unavailable: %w", err)
	}
	return pollShortcut(func() (bool, error) {
		down := true
		for _, key := range shortcut.Keys {
			value, _, _ := keyState.Call(uintptr(key))
			down = down && value&0x8000 != 0
		}
		return down, nil
	}, func() {}, edge, failed), nil
}
