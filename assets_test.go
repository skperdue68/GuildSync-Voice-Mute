package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSecureAssetsSetsHeaders(t *testing.T) {
	called := false
	handler := secureAssets(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		_, _ = w.Write([]byte("ok"))
	}))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/index.html", nil))

	if !strings.Contains(recorder.Header().Get("Content-Security-Policy"), "frame-ancestors 'none'") {
		t.Fatalf("missing frame policy: %q", recorder.Header().Get("Content-Security-Policy"))
	}
	if recorder.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatal("missing nosniff")
	}
	if recorder.Header().Get("Referrer-Policy") != "no-referrer" {
		t.Fatal("missing referrer policy")
	}
	if !called || recorder.Body.String() != "ok" {
		t.Fatal("wrapped handler did not serve the request")
	}
}

// The window must never be blanked by a policy that was not proven in a real
// webview, so the asset server keeps only directives that cannot change how the
// bundle loads or runs.
func TestContentSecurityPolicyDoesNotRestrictLoading(t *testing.T) {
	for _, directive := range []string{"default-src", "script-src", "style-src", "img-src", "font-src", "connect-src", "frame-src", "base-uri"} {
		if strings.Contains(contentSecurityPolicy, directive) {
			t.Fatalf("policy restricts %s: %q", directive, contentSecurityPolicy)
		}
	}
	for _, directive := range []string{"frame-ancestors 'none'", "object-src 'none'", "form-action 'none'"} {
		if !strings.Contains(contentSecurityPolicy, directive) {
			t.Fatalf("policy lost %s: %q", directive, contentSecurityPolicy)
		}
	}
}
