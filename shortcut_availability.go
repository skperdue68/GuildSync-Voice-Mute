package main

// Missing desktop APIs remain unavailable for this app session, independently of login.
type shortcutUnavailableError struct{ reason string }

func (e *shortcutUnavailableError) Error() string { return e.reason }

type ShortcutAvailability struct {
	Available bool   `json:"available"`
	Reason    string `json:"reason"`
}

func (a *App) GetShortcutAvailability() ShortcutAvailability {
	a.shortcutMu.Lock()
	defer a.shortcutMu.Unlock()
	return ShortcutAvailability{Available: a.shortcutUnavailable == nil, Reason: a.shortcutUnavailableReason()}
}
func (a *App) shortcutUnavailableReason() string {
	if a.shortcutUnavailable != nil {
		return a.shortcutUnavailable.Error()
	}
	return ""
}
