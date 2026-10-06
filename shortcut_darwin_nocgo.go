//go:build darwin && !cgo

package main

import "fmt"

func startShortcut(shortcut Shortcut, edge func(bool), failed func(error)) (func(), error) {
	return nil, fmt.Errorf("This build lacks native macOS shortcut support.")
}
