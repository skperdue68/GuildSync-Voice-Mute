package main

import (
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func TestVerificationCannotUndoLogout(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.json")
	s := Session{LoggedIn: true, Allowed: true, Token: "test", ExpiresAt: time.Now().Add(time.Hour), AuthServerURL: backendURL, SocketURL: backendURL}
	if err := writeJSON(path, s); err != nil {
		t.Fatal(err)
	}
	started := make(chan struct{})
	resume := make(chan struct{})
	result := make(chan error, 1)
	a := &App{sessionPath: path, verifySession: func(Session) error { close(started); <-resume; return nil }}
	go func() { _, err := a.GetSession(); result <- err }()
	<-started
	if err := a.Logout(); err != nil {
		t.Fatal(err)
	}
	close(resume)
	if err := <-result; err == nil {
		t.Fatal("stale verification accepted")
	}
	if a.session.LoggedIn {
		t.Fatal("logout overwritten")
	}
}
func TestPollingFailureIsReported(t *testing.T) {
	failed := make(chan error, 1)
	stop := pollShortcut(func() (bool, error) { return false, errors.New("permission revoked") }, func() {}, func(bool) {}, func(err error) { failed <- err })
	defer stop()
	select {
	case err := <-failed:
		if err.Error() != "permission revoked" {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("native error lost")
	}
}
