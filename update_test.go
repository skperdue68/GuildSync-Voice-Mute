package main

import (
	"net/http"
	"net/http/httptest"
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
		w.Write([]byte(`{"tag_name":"v9.9.9","html_url":"https://github.com/skperdue68/GuildSync-Voice-Mute/releases/tag/v9.9.9","body":"Fixes mutes."}`))
	}))
	defer server.Close()
	info, err := (&App{updateURL: server.URL}).CheckForUpdate()
	if err != nil {
		t.Fatal(err)
	}
	if !info.Available || info.Latest != "9.9.9" || info.PageURL == updatePageURL {
		t.Fatalf("%+v", info)
	}
	if info.Notes != "Fixes mutes." || info.Current != productVersion() {
		t.Fatalf("%+v", info)
	}
}

func TestCheckForUpdateIgnoresSameAndPrerelease(t *testing.T) {
	for _, body := range []string{
		`{"tag_name":"v` + productVersion() + `","html_url":"https://github.com/skperdue68/GuildSync-Voice-Mute/releases"}`,
		`{"tag_name":"v99.0.0","prerelease":true,"html_url":"https://github.com/skperdue68/GuildSync-Voice-Mute/releases"}`,
	} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(body))
		}))
		info, err := (&App{updateURL: server.URL}).CheckForUpdate()
		server.Close()
		if err != nil {
			t.Fatal(err)
		}
		if info.Available {
			t.Fatalf("offered %+v", info)
		}
		if info.Message == "" {
			t.Fatalf("no message for %+v", info)
		}
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
	info := UpdateInfo{PageURL: "file:///etc/passwd"}
	if usableReleasePage(info.PageURL) {
		t.Fatal("accepted a non-release URL")
	}
	if err := (&App{}).OpenReleasePage("https://example.com/evil.zip"); err == nil {
		t.Fatal("opened a foreign URL")
	}
	if !usableReleasePage(updatePageURL) || !usableReleasePage(updatePagePrefix+"/tag/v1.0.0") {
		t.Fatal("rejected a release URL")
	}
	if err := (&App{}).OpenReleasePage(updatePageURL); err == nil {
		t.Fatal("missing application reported success")
	}
}
