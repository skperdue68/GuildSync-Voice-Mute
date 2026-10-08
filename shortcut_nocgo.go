//go:build (darwin || linux) && !cgo

package main

func startX11(shortcut Shortcut, edge func(bool), failed func(error)) (func(), error) {
	return nil, &shortcutUnavailableError{"Mute is unavailable for this Linux version/session: this build lacks native X11 shortcut support."}
}
