package main

import "net/http"

// The window renders only its own bundle, so this policy deliberately does not
// allowlist scripts, styles or network origins. Some platforms load the bundle
// from the custom wails:// scheme rather than an HTTP origin, where 'self'
// matching, inline script and frame handling differ per engine, and none of
// that can be exercised without a real webview. A restrictive policy that turns
// out wrong fails closed and shows a blank window, so only directives that
// cannot affect how the bundle loads or runs are sent.
const contentSecurityPolicy = "frame-ancestors 'none'; object-src 'none'; form-action 'none'"

func secureAssets(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", contentSecurityPolicy)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		handler.ServeHTTP(w, r)
	})
}
