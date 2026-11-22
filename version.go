package main

import (
	_ "embed"
	"runtime"
	"strings"
)

// build/config.yml is the single source of truth for the product metadata that
// both the Wails CLI and this binary read.
//
//go:embed build/config.yml
var productMetadata []byte

// productVersion reads info.version without pulling a YAML dependency into the
// binary. The top-level `version: '3'` is the config schema version, so the
// search starts after the info block instead of matching the first "version:".
func productVersion() string {
	text := string(productMetadata)
	start := strings.Index(text, "\ninfo:")
	if start < 0 {
		return "unknown"
	}
	for _, line := range strings.Split(text[start+1:], "\n")[1:] {
		if strings.TrimSpace(line) == "" || strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		if !strings.HasPrefix(line, " ") {
			break
		}
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "version:") {
			continue
		}
		value := strings.Trim(strings.TrimSpace(strings.TrimPrefix(trimmed, "version:")), `"'`)
		if value != "" {
			return value
		}
	}
	return "unknown"
}
func (a *App) GetAppVersion() string { return productVersion() }
func (a *App) GetPlatform() string   { return runtime.GOOS }
