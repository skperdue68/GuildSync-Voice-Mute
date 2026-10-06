//go:build linux

package main

import "os"

func startShortcut(shortcut Shortcut, edge func(bool), failed func(error)) (func(), error) {
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		return startWayland(shortcut, edge, failed)
	}
	return startX11(shortcut, edge, failed)
}
