package main

import (
	_ "embed"
	"encoding/json"
	"runtime"
)

//go:embed wails.json
var productMetadata []byte

func productVersion() string {
	var metadata struct {
		Info struct {
			Version string `json:"productVersion"`
		} `json:"info"`
	}
	if json.Unmarshal(productMetadata, &metadata) != nil || metadata.Info.Version == "" {
		return "unknown"
	}
	return metadata.Info.Version
}
func (a *App) GetAppVersion() string { return productVersion() }
func (a *App) GetPlatform() string   { return runtime.GOOS }
