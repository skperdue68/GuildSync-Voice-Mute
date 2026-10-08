//go:build linux && cgo

package main

/*
#cgo pkg-config: x11
#include <X11/Xlib.h>
#include <X11/keysym.h>
#include <stdlib.h>
static int map_down(char* map, int code){return code > 0 && (map[code >> 3] & (1 << (code & 7)));}
*/
import "C"
import (
	"fmt"
	"strings"
	"unsafe"
)

func startX11(shortcut Shortcut, edge func(bool), failed func(error)) (func(), error) {
	display := C.XOpenDisplay(nil)
	if display == nil {
		return nil, &shortcutUnavailableError{"Mute is unavailable for this Linux version/session: no X11 display is available."}
	}
	codes := [][]int{}
	for _, key := range shortcut.Keys {
		names := []string{}
		switch key {
		case 8, 9, 13, 32, 33, 34, 35, 36, 37, 38, 39, 40, 45, 46:
			names = []string{map[int]string{8: "BackSpace", 9: "Tab", 13: "Return", 32: "space", 33: "Prior", 34: "Next", 35: "End", 36: "Home", 37: "Left", 38: "Up", 39: "Right", 40: "Down", 45: "Insert", 46: "Delete"}[key]}
		case 17:
			names = []string{"Control_L", "Control_R"}
		case 18:
			names = []string{"Alt_L", "Alt_R"}
		case 16:
			names = []string{"Shift_L", "Shift_R"}
		default:
			if key >= 112 {
				names = []string{fmt.Sprintf("F%d", key-111)}
			} else {
				names = []string{strings.ToLower(string(rune(key)))}
			}
		}
		group := []int{}
		for _, name := range names {
			str := C.CString(name)
			sym := C.XStringToKeysym(str)
			C.free(unsafe.Pointer(str))
			code := int(C.XKeysymToKeycode(display, sym))
			if code != 0 {
				group = append(group, code)
			}
		}
		if len(group) == 0 {
			C.XCloseDisplay(display)
			return nil, fmt.Errorf("The selected shortcut is unavailable on this keyboard layout.")
		}
		codes = append(codes, group)
	}
	return pollShortcut(func() (bool, error) {
		var keymap [32]C.char
		C.XQueryKeymap(display, &keymap[0])
		down := true
		for _, group := range codes {
			anyDown := false
			for _, code := range group {
				anyDown = anyDown || C.map_down(&keymap[0], C.int(code)) != 0
			}
			down = down && anyDown
		}
		return down, nil
	}, func() { C.XCloseDisplay(display) }, edge, failed), nil
}
