package main

import "testing"

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
