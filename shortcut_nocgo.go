//go:build (darwin || linux) && !cgo

package main

import "fmt"

func startX11(shortcut Shortcut, edge func(bool), failed func(error)) (func(), error) {
	return nil, fmt.Errorf("This build lacks native X11 shortcut support.")
}
