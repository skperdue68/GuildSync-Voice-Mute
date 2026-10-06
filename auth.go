package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const backendURL = "https://guildsync.perdues.me"
const redirectURI = "http://127.0.0.1:53682/callback"
const discordClientID = "1511276915309281370"

type User struct {
	DiscordUserID string `json:"discord_user_id"`
	Username      string `json:"username"`
	DisplayName   string `json:"display_name"`
	Role          string `json:"role"`
}
type Session struct {
	LoggedIn      bool      `json:"logged_in"`
	Allowed       bool      `json:"allowed"`
	Token         string    `json:"token,omitempty"`
	ExpiresAt     time.Time `json:"expires_at"`
	User          User      `json:"user"`
	SocketURL     string    `json:"socket_url"`
	AuthServerURL string    `json:"auth_server_url"`
	StatusMessage string    `json:"status_message"`
}

func randomToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}
func (a *App) consumeState(value string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	if value == "" || a.loginState == "" || subtle.ConstantTimeCompare([]byte(value), []byte(a.loginState)) != 1 {
		return false
	}
	a.loginState = ""
	return true
}
func (a *App) StartDiscordLogin() error {
	a.mu.Lock()
	a.authGeneration++
	a.mu.Unlock()
	a.stopOAuth()
	listener, err := net.Listen("tcp", "127.0.0.1:53682")
	if err != nil {
		return fmt.Errorf("Discord login callback port 53682 is unavailable. Finish or close the other GuildSync login and try again: %w", err)
	}
	state := randomToken()
	mux := http.NewServeMux()
	server := &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 30 * time.Second}
	a.mu.Lock()
	a.loginState = state
	a.oauthServer = server
	a.mu.Unlock()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", 405)
			return
		}
		if !a.consumeState(r.URL.Query().Get("state")) {
			http.Error(w, "Login security state did not match. Restart login from the companion.", 400)
			return
		}
		finish := func(message string, status int) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(status)
			fmt.Fprintf(w, "<h1>GuildSync Voice Mute</h1><p>%s</p><p>You can close this tab and return to the application.</p>", html.EscapeString(message))
			if status != 200 {
				a.emit("login-error", message)
			}
			go a.stopOAuthServer(server)
		}
		if r.URL.Query().Get("error") != "" {
			finish("Discord login was cancelled or denied.", 400)
			return
		}
		code := r.URL.Query().Get("code")
		if code == "" {
			finish("Discord did not return a login code.", 400)
			return
		}
		s, err := exchangeCode(backendURL, code)
		if err != nil {
			finish(err.Error(), 400)
			return
		}
		if !s.Allowed {
			finish(s.StatusMessage, 403)
			return
		}
		a.mu.Lock()
		if a.oauthServer != server {
			a.mu.Unlock()
			finish("Login was cancelled. Please start again.", 400)
			return
		}
		path, err := dataPath("session.json")
		if err == nil {
			err = writeJSON(path, s)
		}
		if err != nil {
			a.mu.Unlock()
			finish("Unable to save login: "+err.Error(), 500)
			return
		}
		a.session = s
		a.authGeneration++
		a.mu.Unlock()
		a.emit("login-complete", s)
		finish("Login complete.", 200)
	})
	go func() {
		if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			a.emit("login-error", "Local login callback failed.")
		}
	}()
	// Bound abandoned login listeners; do not close a later login's listener.
	go func() { time.Sleep(5 * time.Minute); a.stopOAuthServer(server) }()
	params := url.Values{"response_type": {"code"}, "client_id": {discordClientID}, "redirect_uri": {redirectURI}, "scope": {"identify email"}, "state": {state}, "prompt": {"consent"}}
	runtime.BrowserOpenURL(a.ctx, "https://discord.com/oauth2/authorize?"+params.Encode())
	return nil
}
func (a *App) stopOAuthServer(server *http.Server) {
	a.mu.Lock()
	if a.oauthServer != server {
		a.mu.Unlock()
		return
	}
	a.oauthServer = nil
	a.loginState = ""
	a.mu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = server.Shutdown(ctx)
}
func (a *App) stopOAuth() {
	a.mu.Lock()
	server := a.oauthServer
	a.mu.Unlock()
	if server != nil {
		a.stopOAuthServer(server)
	}
}
func exchangeCode(base, code string) (Session, error) {
	data, _ := json.Marshal(map[string]string{"code": code, "redirect_uri": redirectURI})
	response, err := (&http.Client{Timeout: 20 * time.Second}).Post(base+"/api/auth/discord/desktop-token", "application/json", bytes.NewReader(data))
	if err != nil {
		return Session{}, errors.New("Could not contact the GuildSync login service.")
	}
	defer response.Body.Close()
	var body struct {
		OK        bool      `json:"ok"`
		Allowed   bool      `json:"allowed"`
		Token     string    `json:"token"`
		ExpiresAt time.Time `json:"expires_at"`
		User      User      `json:"user"`
		Message   string    `json:"message"`
		Error     string    `json:"error"`
	}
	if err = json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&body); err != nil {
		return Session{}, errors.New("GuildSync returned an invalid login response.")
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 || !body.OK {
		return Session{}, errors.New("GuildSync rejected the login. " + body.Error)
	}
	if !body.Allowed {
		return Session{StatusMessage: body.Message}, nil
	}
	if body.Token == "" || (!body.ExpiresAt.IsZero() && !body.ExpiresAt.After(time.Now())) {
		return Session{}, errors.New("GuildSync returned an invalid or expired session.")
	}
	return Session{LoggedIn: true, Allowed: true, Token: body.Token, ExpiresAt: body.ExpiresAt, User: body.User, SocketURL: base, AuthServerURL: base, StatusMessage: body.Message}, nil
}
func validateSession(s Session) error {
	if !s.LoggedIn || !s.Allowed || s.Token == "" || (!s.ExpiresAt.IsZero() && !s.ExpiresAt.After(time.Now())) {
		return errors.New("Please sign in with Discord.")
	}
	req, err := http.NewRequest(http.MethodGet, s.AuthServerURL+"/api/auth/session", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+s.Token)
	resp, err := (&http.Client{Timeout: 5 * time.Second}).Do(req)
	if err != nil {
		return errors.New("Unable to verify your GuildSync account; reconnect to try again.")
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return errors.New("Your GuildSync session is unavailable or has ended. Sign in again.")
	}
	return nil
}
func (a *App) GetSession() (Session, error) {
	a.mu.Lock()
	generation := a.authGeneration
	a.mu.Unlock()
	path, err := a.sessionFile()
	if err != nil {
		return Session{}, err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return Session{}, nil
	}
	if err != nil {
		return Session{}, err
	}
	var s Session
	if json.Unmarshal(data, &s) != nil {
		return Session{}, errors.New("Saved login is invalid. Sign in again.")
	}
	// Prevent a tampered local file from sending a bearer token to another host.
	if s.AuthServerURL != backendURL || s.SocketURL != backendURL {
		return Session{}, errors.New("Saved login endpoint is invalid. Sign in again.")
	}
	verify := a.verifySession
	if verify == nil {
		verify = validateSession
	}
	err = verify(s)
	a.mu.Lock()
	if generation != a.authGeneration {
		a.mu.Unlock()
		return Session{}, errors.New("Login changed while verifying the session.")
	}
	if err != nil {
		a.session = Session{}
		a.mu.Unlock()
		_ = a.SetShortcutActive(false)
		return Session{}, err
	}
	a.session = s
	a.mu.Unlock()
	return s, nil
}
func (a *App) Logout() error {
	a.mu.Lock()
	a.authGeneration++
	a.mu.Unlock()
	_ = a.SetShortcutActive(false)
	a.stopOAuth()
	a.mu.Lock()
	s := a.session
	a.session = Session{}
	a.mu.Unlock()
	path, err := a.sessionFile()
	if err != nil {
		return err
	}
	if err = os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	if s.Token != "" {
		req, _ := http.NewRequest(http.MethodPost, backendURL+"/api/auth/logout", nil)
		req.Header.Set("Authorization", "Bearer "+s.Token)
		resp, err := (&http.Client{Timeout: 5 * time.Second}).Do(req)
		if err == nil {
			resp.Body.Close()
		}
	}
	return nil
}
func (a *App) sessionFile() (string, error) {
	if a.sessionPath != "" {
		return a.sessionPath, nil
	}
	return dataPath("session.json")
}
