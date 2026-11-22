package main

import (
	"embed"

	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	service := NewApp()
	app := application.New(application.Options{
		Name:        "GuildSync Voice Mute",
		Description: "GuildSync voice-channel mute companion",
		Services:    []application.Service{application.NewService(service)},
		Assets:      application.AssetOptions{Handler: application.AssetFileServerFS(assets)},
		Mac:         application.MacOptions{ApplicationShouldTerminateAfterLastWindowClosed: true},
		Linux:       application.LinuxOptions{ProgramName: "GuildSync Voice Mute"},
	})
	service.app = app
	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:      "main",
		Title:     "GuildSync Voice Mute v" + productVersion(),
		URL:       "/",
		Width:     480,
		Height:    650,
		MinWidth:  400,
		MinHeight: 480,
	})
	err := app.Run()
	if err != nil {
		println(err.Error())
	}
}
