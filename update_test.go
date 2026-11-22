package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCompareVersions(t *testing.T) {
	for _, tc := range []struct {
		a, b string
		want int
	}{
		{"1.2.0", "1.1.9", 1},
		{"v1.0.0", "1.0.0", 0},
		{"1.0.0-beta.1", "1.0.0", 0},
		{"0.9", "1.0", -1},
		{"2", "1.9.9", 1},
		{"nonsense", "1.0.0", 0},
	} {
		if got := compareVersions(tc.a, tc.b); got != tc.want {
			t.Errorf("compareVersions(%q, %q) = %d, want %d", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestCheckForUpdateFindsNewerRelease(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") == "" {
			t.Error("missing user agent")
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"tag_name":"v9.9.9","html_url":"` + updateReleaseURL + `/tag/v9.9.9"}`))
	}))
	defer server.Close()
	info, err := (&App{updateURL: server.URL}).CheckForUpdate()
	if err != nil {
		t.Fatal(err)
	}
	if !info.Available || info.PageURL != updateReleaseURL+"/tag/v9.9.9" {
		t.Fatalf("%+v", info)
	}
	if !strings.Contains(info.Message, "9.9.9") {
		t.Fatalf("message does not name the version: %+v", info)
	}
}

func TestCheckForUpdateDoesNotOfferTheRunningVersion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"tag_name":"v` + productVersion() + `","html_url":"` + updateReleaseURL + `"}`))
	}))
	defer server.Close()
	info, err := (&App{updateURL: server.URL}).CheckForUpdate()
	if err != nil {
		t.Fatal(err)
	}
	if info.Available || info.Message == "" || info.PageURL != updateReleaseURL {
		t.Fatalf("%+v", info)
	}
}

func TestCheckForUpdateReportsHTTPFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "rate limited", http.StatusForbidden)
	}))
	defer server.Close()
	if _, err := (&App{updateURL: server.URL}).CheckForUpdate(); err == nil {
		t.Fatal("HTTP failure reported as success")
	}
}

func TestReleasePageIsValidated(t *testing.T) {
	if usableReleasePage("file:///etc/passwd") {
		t.Fatal("accepted a non-release URL")
	}
	if !usableReleasePage(updateReleaseURL) || !usableReleasePage(updateReleaseURL+"/tag/v1.0.0") {
		t.Fatal("rejected a release URL")
	}
	if !strings.HasPrefix(updateAPIURL, "https://api.github.com/repos/"+updateRepo+"/") {
		t.Fatalf("update endpoint left the repository: %s", updateAPIURL)
	}
	if err := (&App{}).OpenReleasePage("https://example.com/evil.zip"); err == nil {
		t.Fatal("opened a foreign URL")
	}
	if err := (&App{}).OpenReleasePage(updateReleaseURL); err == nil {
		t.Fatal("missing application reported success")
	}
}
