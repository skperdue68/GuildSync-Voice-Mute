//go:build linux

package main

import (
	"context"
	"fmt"
	"github.com/godbus/dbus/v5"
	"strings"
	"sync"
	"time"
)

const portalName = "org.freedesktop.portal.Desktop"
const portalPath = dbus.ObjectPath("/org/freedesktop/portal/desktop")
const shortcutsInterface = "org.freedesktop.portal.GlobalShortcuts"

type portalShortcut struct {
	ID      string
	Options map[string]dbus.Variant
}

func startWayland(shortcut Shortcut, edge func(bool), failed func(error)) (func(), error) {
	actionKeys := 0
	for _, key := range shortcut.Keys {
		if key != 16 && key != 17 && key != 18 {
			actionKeys++
		}
	}
	if actionKeys != 1 {
		return nil, fmt.Errorf("This Wayland shortcut portal requires one ordinary key, optionally with Ctrl, Alt or Shift. Arbitrary multi-key holds are supported on Windows, macOS and X11.")
	}
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return nil, &shortcutUnavailableError{fmt.Sprintf("Mute is unavailable for this Linux version/session: Wayland shortcut portal is unavailable: %v", err)}
	}
	success := false
	defer func() {
		if !success {
			conn.Close()
		}
	}()
	var version uint32
	if err = conn.Object(portalName, portalPath).Call("org.freedesktop.DBus.Properties.Get", 0, shortcutsInterface, "version").Store(new(dbus.Variant)); err != nil {
		return nil, &shortcutUnavailableError{fmt.Sprintf("Mute is unavailable for this Linux version/session: this desktop does not provide the GlobalShortcuts portal: %v", err)}
	}
	_ = version
	signals := make(chan *dbus.Signal, 32)
	conn.Signal(signals)
	if err = conn.AddMatchSignal(dbus.WithMatchSender(portalName), dbus.WithMatchInterface("org.freedesktop.portal.Request")); err != nil {
		return nil, err
	}
	if err = conn.AddMatchSignal(dbus.WithMatchSender(portalName), dbus.WithMatchInterface(shortcutsInterface)); err != nil {
		return nil, err
	}
	token := strings.ReplaceAll(randomToken(), "-", "_")
	opts := map[string]dbus.Variant{"handle_token": dbus.MakeVariant("r" + token), "session_handle_token": dbus.MakeVariant("s" + token)}
	response, err := portalRequest(conn, signals, shortcutsInterface+".CreateSession", opts)
	if err != nil {
		return nil, err
	}
	handle, ok := response["session_handle"].Value().(string)
	if !ok || !dbus.ObjectPath(handle).IsValid() {
		return nil, fmt.Errorf("Shortcut portal returned an invalid session.")
	}
	session := dbus.ObjectPath(handle)
	defer func() {
		if !success {
			conn.Object(portalName, session).Call("org.freedesktop.portal.Session.Close", 0)
		}
	}()
	opts = map[string]dbus.Variant{"handle_token": dbus.MakeVariant("b" + token)}
	trigger := strings.ReplaceAll(shortcut.Label, "Ctrl", "CTRL")
	response, err = portalRequest(conn, signals, shortcutsInterface+".BindShortcuts", session, []portalShortcut{{"mute", map[string]dbus.Variant{"description": dbus.MakeVariant("Hold to mute lower ranks in your Discord voice channel"), "preferred_trigger": dbus.MakeVariant(trigger)}}}, "", opts)
	if err != nil {
		return nil, err
	}
	var bound []portalShortcut
	if err = dbus.Store([]interface{}{response["shortcuts"].Value()}, &bound); err != nil || len(bound) == 0 {
		return nil, fmt.Errorf("No shortcut was approved. Enable again and choose a shortcut in the desktop dialog.")
	}
	done := make(chan struct{})
	var once sync.Once
	success = true
	// Only future portal activation events count. Binding does not replay a key state.
	edge(false)
	go func() {
		for {
			select {
			case <-done:
				return
			case signal, open := <-signals:
				if !open {
					edge(false)
					return
				}
				if signal == nil || len(signal.Body) < 2 {
					continue
				}
				if signal.Body[0] != session || signal.Body[1] != "mute" {
					continue
				}
				switch signal.Name {
				case shortcutsInterface + ".Activated":
					edge(true)
				case shortcutsInterface + ".Deactivated":
					edge(false)
				}
			}
		}
	}()
	return func() {
		once.Do(func() {
			close(done)
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				defer cancel()
				conn.Object(portalName, session).CallWithContext(ctx, "org.freedesktop.portal.Session.Close", 0)
				conn.Close()
			}()
		})
	}, nil
}
func portalRequest(conn *dbus.Conn, signals <-chan *dbus.Signal, method string, args ...interface{}) (map[string]dbus.Variant, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	var request dbus.ObjectPath
	if err := conn.Object(portalName, portalPath).CallWithContext(ctx, method, 0, args...).Store(&request); err != nil {
		return nil, fmt.Errorf("Shortcut portal request failed: %w", err)
	}
	for {
		select {
		case <-ctx.Done():
			conn.Object(portalName, request).Call("org.freedesktop.portal.Request.Close", 0)
			return nil, fmt.Errorf("Shortcut approval timed out. Try enabling it again.")
		case signal, open := <-signals:
			if !open {
				return nil, fmt.Errorf("Shortcut portal disconnected.")
			}
			if signal.Path != request || signal.Name != "org.freedesktop.portal.Request.Response" {
				continue
			}
			var code uint32
			var results map[string]dbus.Variant
			if err := dbus.Store(signal.Body, &code, &results); err != nil {
				return nil, err
			}
			if code != 0 {
				return nil, fmt.Errorf("Shortcut permission was declined or unavailable.")
			}
			return results, nil
		}
	}
}
