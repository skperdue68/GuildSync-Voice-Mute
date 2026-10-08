package main

import (
	"embed"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	a := NewApp()
	err := wails.Run(&options.App{Title: "GuildSync Voice Mute v" + productVersion(), Width: 480, Height: 650, MinWidth: 400, MinHeight: 480, AssetServer: &assetserver.Options{Assets: assets}, OnStartup: a.startup, OnShutdown: a.shutdown, Bind: []interface{}{a}})
	if err != nil {
		println(err.Error())
	}
}
