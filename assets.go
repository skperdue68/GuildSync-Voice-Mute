package main

import "net/http"

// The window only ever renders its own bundle. The policy keeps that bundle from
// loading remote code and limits network access to the GuildSync backend the
// socket already talks to. Wails injects its runtime as an inline script, so
// inline scripts and styles must stay allowed.
const contentSecurityPolicy = "default-src 'self'; " +
	"script-src 'self' 'unsafe-inline'; " +
	"style-src 'self' 'unsafe-inline'; " +
	"img-src 'self' data:; " +
	"connect-src 'self' https://guildsync.perdues.me wss://guildsync.perdues.me; " +
	"object-src 'none'; base-uri 'none'; form-action 'none'; frame-ancestors 'none'"

func secureAssets(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", contentSecurityPolicy)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		handler.ServeHTTP(w, r)
	})
}
