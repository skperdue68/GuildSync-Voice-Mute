//go:build linux

package main

import (
	"context"
	"time"

	"github.com/godbus/dbus/v5"
)

// trayHostAvailable reports whether a panel is actually showing notification
// icons. A watcher service alone is not enough, so the registered-host property
// is checked without auto-starting anything.
func trayHostAvailable() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	conn, err := dbus.ConnectSessionBus(dbus.WithContext(ctx))
	if err != nil {
		return false
	}
	defer conn.Close()
	var registered dbus.Variant
	err = conn.Object("org.kde.StatusNotifierWatcher", "/StatusNotifierWatcher").CallWithContext(ctx,
		"org.freedesktop.DBus.Properties.Get", dbus.FlagNoAutoStart,
		"org.kde.StatusNotifierWatcher", "IsStatusNotifierHostRegistered").Store(&registered)
	available, _ := registered.Value().(bool)
	return err == nil && available
}
