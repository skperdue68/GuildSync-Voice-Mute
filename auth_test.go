package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestOAuthStateIsOneUse(t *testing.T) {
	a := &App{loginState: "expected"}
	if a.consumeState("wrong") {
		t.Fatal("state mismatch accepted")
	}
	if !a.consumeState("expected") || a.consumeState("expected") {
		t.Fatal("state not one-use")
	}
}
func TestSessionRejectedOnRevocation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(401) }))
	defer server.Close()
	s := Session{LoggedIn: true, Allowed: true, Token: "test", ExpiresAt: time.Now().Add(time.Hour), AuthServerURL: server.URL}
	if err := validateSession(s); err == nil {
		t.Fatal("revoked session accepted")
	}
	s.ExpiresAt = time.Now().Add(-time.Hour)
	if err := validateSession(s); err == nil {
		t.Fatal("expired session accepted")
	}
}
func TestPendingLoginDoesNotPersistToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok":true,"allowed":false,"message":"Pending approval"}`))
	}))
	defer server.Close()
	s, err := exchangeCode(server.URL, "code")
	if err != nil || s.Allowed || s.LoggedIn || s.Token != "" {
		t.Fatalf("%+v %v", s, err)
	}
}
