package main

import (
	"embed"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	service := NewApp()
	app := application.New(application.Options{
		Name:        "GuildSync Voice Mute",
		Description: "GuildSync voice-channel mute companion",
		Services:    []application.Service{application.NewService(service)},
		Assets:      application.AssetOptions{Handler: secureAssets(application.AssetFileServerFS(assets))},
		Mac:         application.MacOptions{ApplicationShouldTerminateAfterLastWindowClosed: true},
		Linux:       application.LinuxOptions{ProgramName: "GuildSync Voice Mute"},
		// One process owns the OAuth callback port and the global shortcut, so a
		// second launch focuses the running window instead of failing at login or
		// double-driving mute requests.
		SingleInstance: &application.SingleInstanceOptions{
			UniqueID:               "me.perdues.guildsync.voicemute",
			OnSecondInstanceLaunch: func(application.SecondInstanceData) { service.ShowWindow() },
		},
	})
	service.app = app
	window := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:      "main",
		Title:     "GuildSync Voice Mute v" + productVersion(),
		URL:       "/",
		Width:     480,
		Height:    650,
		MinWidth:  400,
		MinHeight: 480,
	})
	service.setWindow(window)
	// A hook, not a listener: only a hook can cancel the close.
	window.RegisterHook(events.Common.WindowClosing, func(event *application.WindowEvent) {
		if service.hideOnClose() {
			window.Hide()
			event.Cancel()
		}
	})
	service.startTray()
	// Look for a release on the clock rather than in the window, so an update is
	// offered even when the window spends the day hidden in the tray.
	service.startUpdateWatcher()
	err := app.Run()
	if err != nil {
		println(err.Error())
	}
}
