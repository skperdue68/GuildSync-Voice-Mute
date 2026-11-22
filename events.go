package main

import "github.com/wailsapp/wails/v3/pkg/application"

// Custom events are registered with their payload type so the binding generator
// documents them and the event manager rejects a mismatched payload. Keeping the
// names in constants means a rename here is a single edit shared by the emit and
// the registration, instead of a string that silently stops matching the UI.
const (
	eventVoiceShortcutEdge   = "voice-shortcut-edge"
	eventVoiceShortcutStatus = "voice-shortcut-status"
	eventLoginComplete       = "login-complete"
	eventLoginError          = "login-error"
	eventShortcutError       = "shortcut-error"
)

func init() {
	application.RegisterEvent[bool](eventVoiceShortcutEdge)
	application.RegisterEvent[string](eventVoiceShortcutStatus)
	application.RegisterEvent[Session](eventLoginComplete)
	application.RegisterEvent[string](eventLoginError)
	application.RegisterEvent[string](eventShortcutError)
}
