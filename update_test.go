package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
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

func TestPickUpdateAssetMatchesEachPlatform(t *testing.T) {
	assets := []updateAsset{
		{Name: "GuildSync-Voice-Mute-2.1.0-Linux-amd64.deb", URL: updateAssetBase + "v2.1.0/GuildSync-Voice-Mute-2.1.0-Linux-amd64.deb"},
		{Name: "GuildSync-Voice-Mute-2.1.0-Linux-arm64.deb", URL: updateAssetBase + "v2.1.0/GuildSync-Voice-Mute-2.1.0-Linux-arm64.deb"},
		{Name: "GuildSync-Voice-Mute-2.1.0-macOS.dmg", URL: updateAssetBase + "v2.1.0/GuildSync-Voice-Mute-2.1.0-macOS.dmg"},
		{Name: "GuildSync-Voice-Mute-2.1.0-Windows-Setup.exe", URL: updateAssetBase + "v2.1.0/GuildSync-Voice-Mute-2.1.0-Windows-Setup.exe"},
	}
	for _, tc := range []struct{ goos, goarch, want string }{
		{"windows", "amd64", "GuildSync-Voice-Mute-2.1.0-Windows-Setup.exe"},
		// The setup is x64-compatible, so an arm64 PC runs the same one.
		{"windows", "arm64", "GuildSync-Voice-Mute-2.1.0-Windows-Setup.exe"},
		// The disk image holds a universal bundle.
		{"darwin", "arm64", "GuildSync-Voice-Mute-2.1.0-macOS.dmg"},
		{"darwin", "amd64", "GuildSync-Voice-Mute-2.1.0-macOS.dmg"},
		// A Debian package is built for one machine only.
		{"linux", "amd64", "GuildSync-Voice-Mute-2.1.0-Linux-amd64.deb"},
		{"linux", "arm64", "GuildSync-Voice-Mute-2.1.0-Linux-arm64.deb"},
	} {
		asset, ok := pickUpdateAsset(assets, tc.goos, tc.goarch, updateAssetBase)
		if !ok || asset.Name != tc.want {
			t.Errorf("pickUpdateAsset(%s/%s) = %q, %v; want %q", tc.goos, tc.goarch, asset.Name, ok, tc.want)
		}
	}
}

func TestPickUpdateAssetRefusesAnotherArchitecture(t *testing.T) {
	// A release built only on amd64 must not hand an arm64 machine a package
	// dpkg will reject.
	assets := []updateAsset{{
		Name: "GuildSync-Voice-Mute-2.1.0-Linux-amd64.deb",
		URL:  updateAssetBase + "v2.1.0/GuildSync-Voice-Mute-2.1.0-Linux-amd64.deb",
	}}
	if asset, ok := pickUpdateAsset(assets, "linux", "arm64", updateAssetBase); ok {
		t.Fatalf("an arm64 machine picked %q", asset.Name)
	}
	if asset, ok := pickUpdateAsset(assets, "linux", "amd64", updateAssetBase); !ok || asset.Name != assets[0].Name {
		t.Fatalf("an amd64 machine picked %q, %v", asset.Name, ok)
	}
}

func TestUpdateSuffixNamesTheDebianArchitecture(t *testing.T) {
	for goarch, want := range map[string]string{
		"amd64":   "-Linux-amd64.deb",
		"arm64":   "-Linux-arm64.deb",
		"386":     "-Linux-i386.deb",
		"arm":     "-Linux-armhf.deb",
		"ppc64le": "-Linux-ppc64el.deb",
	} {
		if got := updateSuffix("linux", goarch); got != want {
			t.Errorf("updateSuffix(linux, %q) = %q, want %q", goarch, got, want)
		}
	}
	if got := updateSuffix("linux", "wasm"); got != "" {
		t.Errorf("updateSuffix(linux, wasm) = %q, want no installer", got)
	}
	if got := updateSuffix("windows", "arm64"); got != "-Windows-Setup.exe" {
		t.Errorf("updateSuffix(windows, arm64) = %q", got)
	}
}

func TestPickUpdateAssetRefusesAnotherHostOrPlatform(t *testing.T) {
	assets := []updateAsset{{Name: "GuildSync-Voice-Mute-2.1.0-Linux-amd64.deb", URL: "https://example.com/GuildSync-Voice-Mute-2.1.0-Linux-amd64.deb"}}
	if _, ok := pickUpdateAsset(assets, "linux", "amd64", updateAssetBase); ok {
		t.Fatal("picked an asset from another host")
	}
	if _, ok := pickUpdateAsset(assets, "plan9", "amd64", updateAssetBase); ok {
		t.Fatal("picked an asset for an unsupported platform")
	}
}

func TestDigestMatches(t *testing.T) {
	digest := "sha256:abcd"
	if !digestMatches("", "abcd") {
		t.Fatal("an unpublished digest rejected the download")
	}
	if !digestMatches(digest, "ABCD") {
		t.Fatal("a matching digest was rejected")
	}
	if digestMatches(digest, "abce") || digestMatches("md5:abcd", "abcd") {
		t.Fatal("a wrong or unsupported digest was accepted")
	}
}

func TestFetchUpdateAssetVerifiesTheDigest(t *testing.T) {
	payload := []byte("installer payload")
	sum := sha256.Sum256(payload)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(payload)
	}))
	defer server.Close()
	directory := t.TempDir()
	asset := updateAsset{
		Name:   "GuildSync-Voice-Mute-9.9.9-Linux.deb",
		URL:    server.URL + "/download",
		Digest: "sha256:" + hex.EncodeToString(sum[:]),
	}
	path, err := fetchUpdateAsset(asset, directory)
	if err != nil {
		t.Fatal(err)
	}
	if path != filepath.Join(directory, asset.Name) {
		t.Fatalf("saved to %q", path)
	}
	if contents, err := os.ReadFile(path); err != nil || !bytes.Equal(contents, payload) {
		t.Fatalf("downloaded %q, %v", contents, err)
	}
	asset.Digest = "sha256:" + strings.Repeat("0", 64)
	badDirectory := t.TempDir()
	if _, err := fetchUpdateAsset(asset, badDirectory); err == nil {
		t.Fatal("a checksum mismatch was reported as success")
	}
	if entries, err := os.ReadDir(badDirectory); err != nil || len(entries) != 0 {
		t.Fatalf("a rejected download was left behind: %v, %v", entries, err)
	}
}

func TestCheckForUpdateDownloadsTheInstallersForThisPlatform(t *testing.T) {
	suffix := updateSuffix(runtime.GOOS, runtime.GOARCH)
	if suffix == "" {
		t.Skip("no installer is published for this platform")
	}
	payload := []byte("installer payload")
	sum := sha256.Sum256(payload)
	name := "GuildSync-Voice-Mute-9.9.9" + suffix
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/download/"):
			w.Write(payload)
		case strings.HasPrefix(r.URL.Path, "/latest"):
			fmt.Fprintf(w, `{"tag_name":"v9.9.9","html_url":%q,"assets":[{"name":%q,"browser_download_url":%q,"digest":%q}]}`,
				updateReleaseURL+"/tag/v9.9.9", name, server.URL+"/download/"+name, "sha256:"+hex.EncodeToString(sum[:]))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	if err := os.Mkdir(filepath.Join(home, "Downloads"), 0o755); err != nil {
		t.Fatal(err)
	}
	opened := ""
	info, err := (&App{
		updateURL: server.URL + "/latest",
		assetBase: server.URL + "/download/",
		openPath:  func(path string) error { opened = path; return nil },
	}).CheckForUpdate()
	if err != nil {
		t.Fatal(err)
	}
	if !info.Available || filepath.Base(opened) != name {
		t.Fatalf("%+v opened %q, want %q", info, opened, name)
	}
	if contents, err := os.ReadFile(opened); err != nil || !bytes.Equal(contents, payload) {
		t.Fatalf("downloaded %q, %v", contents, err)
	}
	if !strings.Contains(info.Message, name) || !strings.Contains(info.Message, "opened it") {
		t.Fatalf("message does not report the download: %q", info.Message)
	}
}

func TestCheckForUpdateSaysWhenTheReleaseHasNoInstaller(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"tag_name":"v9.9.9","html_url":"` + updateReleaseURL + `/tag/v9.9.9","assets":[]}`))
	}))
	defer server.Close()
	info, err := (&App{updateURL: server.URL}).CheckForUpdate()
	if err != nil {
		t.Fatal(err)
	}
	if !info.Available || !strings.Contains(info.Message, "no installer for this platform") {
		t.Fatalf("%+v", info)
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
