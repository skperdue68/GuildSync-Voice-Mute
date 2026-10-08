package main

import "testing"

func TestMissingDesktopAPIKeepsAppUsableWithoutRepeatedActivation(t *testing.T) {
	calls := 0
	a := &App{settings: Settings{Enabled: true, Shortcut: "Ctrl+M"}, session: Session{LoggedIn: true, Allowed: true, Token: "test"}}
	a.startListener = func(Shortcut, func(bool), func(error)) (func(), error) {
		calls++
		return nil, &shortcutUnavailableError{"Mute is unavailable for this Linux version/session"}
	}
	for i := 0; i < 3; i++ {
		if a.SetShortcutActive(true) == nil {
			t.Fatal("missing API activated")
		}
	}
	if calls != 1 {
		t.Fatalf("retried missing API %d times", calls)
	}
	state := a.GetShortcutAvailability()
	if state.Available || state.Reason == "" {
		t.Fatal("missing availability reason")
	}
	if err := a.SetShortcutActive(false); err != nil {
		t.Fatal(err)
	}
	if a.GetShortcutSettings().Shortcut != "Ctrl+M" {
		t.Fatal("settings inaccessible")
	}
}

func TestNativeInitialReleaseUnblocksFrontend(t *testing.T) {
	a := &App{active: true, hold: holdState{blocked: true}, listenerGeneration: 1}
	edges := []bool{}
	a.emitEdge = func(down bool) { edges = append(edges, down) }
	a.observeEdge(1, false)
	a.observeEdge(1, true)
	if len(edges) != 2 || edges[0] || !edges[1] {
		t.Fatalf("first press was lost: %v", edges)
	}
	a.listenerGeneration = 2
	a.observeEdge(1, true)
	if len(edges) != 2 {
		t.Fatal("stale listener emitted an edge")
	}
}
