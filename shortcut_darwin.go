//go:build darwin && cgo

package main

/*
#cgo LDFLAGS: -framework ApplicationServices
#include <ApplicationServices/ApplicationServices.h>
static int listen_allowed(void) { return CGPreflightListenEventAccess(); }
static void request_listen(void) { CGRequestListenEventAccess(); }
static int key_down(int key) { return CGEventSourceKeyState(kCGEventSourceStateCombinedSessionState, key); }
*/
import "C"
import "fmt"

func startShortcut(shortcut Shortcut, edge func(bool), failed func(error)) (func(), error) {
	if C.listen_allowed() == 0 {
		C.request_listen()
		return nil, fmt.Errorf("Allow GuildSync Voice Mute in System Settings → Privacy & Security → Input Monitoring, then restart the app and enable the shortcut.")
	}
	codes := map[int][]int{17: {59, 62}, 18: {58, 61}, 16: {56, 60}, 65: {0}, 66: {11}, 67: {8}, 68: {2}, 69: {14}, 70: {3}, 71: {5}, 72: {4}, 73: {34}, 74: {38}, 75: {40}, 76: {37}, 77: {46}, 78: {45}, 79: {31}, 80: {35}, 81: {12}, 82: {15}, 83: {1}, 84: {17}, 85: {32}, 86: {9}, 87: {13}, 88: {7}, 89: {16}, 90: {6}, 48: {29}, 49: {18}, 50: {19}, 51: {20}, 52: {21}, 53: {23}, 54: {22}, 55: {26}, 56: {28}, 57: {25}, 112: {122}, 113: {120}, 114: {99}, 115: {118}, 116: {96}, 117: {97}, 118: {98}, 119: {100}, 120: {101}, 121: {109}, 122: {103}, 123: {111}}
	return pollShortcut(func() (bool, error) {
		if C.listen_allowed() == 0 {
			return false, fmt.Errorf("Input Monitoring permission was revoked")
		}
		down := true
		for _, key := range shortcut.Keys {
			anyDown := false
			for _, code := range codes[key] {
				anyDown = anyDown || C.key_down(C.int(code)) != 0
			}
			down = down && anyDown
		}
		return down, nil
	}, func() {}, edge, failed), nil
}
