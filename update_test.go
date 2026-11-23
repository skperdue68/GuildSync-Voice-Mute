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
	// The portable archives are published next to the installers, so the
	// installer has to win.
	assets := []updateAsset{
		{Name: "GuildSync-Voice-Mute-2.1.0-Linux-x86_64.AppImage", URL: updateAssetBase + "v2.1.0/GuildSync-Voice-Mute-2.1.0-Linux-x86_64.AppImage"},
		{Name: "GuildSync-Voice-Mute-2.1.0-Linux-aarch64.AppImage", URL: updateAssetBase + "v2.1.0/GuildSync-Voice-Mute-2.1.0-Linux-aarch64.AppImage"},
		{Name: "GuildSync-Voice-Mute-2.1.0-macOS.dmg", URL: updateAssetBase + "v2.1.0/GuildSync-Voice-Mute-2.1.0-macOS.dmg"},
		{Name: "GuildSync-Voice-Mute-2.1.0-Windows.zip", URL: updateAssetBase + "v2.1.0/GuildSync-Voice-Mute-2.1.0-Windows.zip"},
		{Name: "GuildSync-Voice-Mute-2.1.0-Windows-Setup.exe", URL: updateAssetBase + "v2.1.0/GuildSync-Voice-Mute-2.1.0-Windows-Setup.exe"},
	}
	for _, tc := range []struct{ goos, goarch, want string }{
		{"windows", "amd64", "GuildSync-Voice-Mute-2.1.0-Windows-Setup.exe"},
		// The setup is x64-compatible, so an arm64 PC runs the same one.
		{"windows", "arm64", "GuildSync-Voice-Mute-2.1.0-Windows-Setup.exe"},
		// The disk image holds a universal bundle.
		{"darwin", "arm64", "GuildSync-Voice-Mute-2.1.0-macOS.dmg"},
		{"darwin", "amd64", "GuildSync-Voice-Mute-2.1.0-macOS.dmg"},
		// An AppImage is built for one machine only.
		{"linux", "amd64", "GuildSync-Voice-Mute-2.1.0-Linux-x86_64.AppImage"},
		{"linux", "arm64", "GuildSync-Voice-Mute-2.1.0-Linux-aarch64.AppImage"},
	} {
		asset, ok := pickUpdateAsset(assets, tc.goos, tc.goarch, updateAssetBase)
		if !ok || asset.Name != tc.want {
			t.Errorf("pickUpdateAsset(%s/%s) = %q, %v; want %q", tc.goos, tc.goarch, asset.Name, ok, tc.want)
		}
	}
}

func TestPickUpdateAssetRefusesAnotherArchitecture(t *testing.T) {
	// A release built only on amd64 must not hand an arm64 machine an image it
	// cannot run.
	assets := []updateAsset{{
		Name: "GuildSync-Voice-Mute-2.1.0-Linux-x86_64.AppImage",
		URL:  updateAssetBase + "v2.1.0/GuildSync-Voice-Mute-2.1.0-Linux-x86_64.AppImage",
	}}
	if asset, ok := pickUpdateAsset(assets, "linux", "arm64", updateAssetBase); ok {
		t.Fatalf("an arm64 machine picked %q", asset.Name)
	}
	if asset, ok := pickUpdateAsset(assets, "linux", "amd64", updateAssetBase); !ok || asset.Name != assets[0].Name {
		t.Fatalf("an amd64 machine picked %q, %v", asset.Name, ok)
	}
}

func TestUpdateSuffixNamesTheLinuxArchitecture(t *testing.T) {
	for goarch, want := range map[string]string{
		"amd64": "-Linux-x86_64.AppImage",
		"arm64": "-Linux-aarch64.AppImage",
	} {
		if got := updateSuffix("linux", goarch); got != want {
			t.Errorf("updateSuffix(linux, %q) = %q, want %q", goarch, got, want)
		}
	}
	// The AppImage generator builds no other Linux architecture, so those
	// machines are offered the release page instead.
	for _, goarch := range []string{"386", "arm", "ppc64le", "wasm"} {
		if got := updateSuffix("linux", goarch); got != "" {
			t.Errorf("updateSuffix(linux, %q) = %q, want no installer", goarch, got)
		}
	}
	if got := updateSuffix("windows", "arm64"); got != "-Windows-Setup.exe" {
		t.Errorf("updateSuffix(windows, arm64) = %q", got)
	}
}

func TestPickUpdateAssetRefusesAnotherHostOrPlatform(t *testing.T) {
	assets := []updateAsset{{Name: "GuildSync-Voice-Mute-2.1.0-Linux-x86_64.AppImage", URL: "https://example.com/GuildSync-Voice-Mute-2.1.0-Linux-x86_64.AppImage"}}
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
		Name:   "GuildSync-Voice-Mute-9.9.9-Linux-x86_64.AppImage",
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

func TestMakeExecutableSetsTheBitOnLinux(t *testing.T) {
	// An AppImage is run as a program, so the saved copy has to be executable or
	// opening it fails with "Permission denied". Windows records no executable
	// bit at all — a chmod there leaves every writable file at 0666 — so this
	// can only be checked on the platform that reads the bit.
	if runtime.GOOS != "linux" {
		t.Skip("only Linux keeps the executable bit in the file mode")
	}
	path := filepath.Join(t.TempDir(), "GuildSync-Voice-Mute-9.9.9-Linux-x86_64.AppImage")
	if err := os.WriteFile(path, []byte("appimage payload"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := makeExecutable(path, "linux"); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != installerMode {
		t.Fatalf("mode = %v, want %v", info.Mode().Perm(), installerMode)
	}
	if contents, err := os.ReadFile(path); err != nil || string(contents) != "appimage payload" {
		t.Fatalf("setting the mode changed the file: %q, %v", contents, err)
	}
	// The bit is set by the download, not by the user, so running it twice must
	// stay harmless.
	if err := makeExecutable(path, "linux"); err != nil {
		t.Fatal(err)
	}
	if info, err := os.Stat(path); err != nil || info.Mode().Perm() != installerMode {
		t.Fatalf("a second pass left mode %v, %v", info.Mode(), err)
	}
}

func TestMakeExecutableLeavesOtherPlatformsAlone(t *testing.T) {
	// Windows runs the setup by name and macOS mounts the disk image, so neither
	// reads the mode and the downloaded file keeps the one it was given.
	for _, goos := range []string{"windows", "darwin", "plan9"} {
		path := filepath.Join(t.TempDir(), "installer")
		if err := os.WriteFile(path, []byte("payload"), 0o644); err != nil {
			t.Fatal(err)
		}
		before, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if err := makeExecutable(path, goos); err != nil {
			t.Fatalf("%s: %v", goos, err)
		}
		after, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if after.Mode() != before.Mode() {
			t.Fatalf("%s changed the mode from %v to %v", goos, before.Mode(), after.Mode())
		}
	}
}

func TestMakeExecutableReportsAFailure(t *testing.T) {
	// A download that cannot be made runnable must not be reported as ready.
	if err := makeExecutable(filepath.Join(t.TempDir(), "missing.AppImage"), "linux"); err == nil {
		t.Fatal("a missing file was reported as executable")
	}
}

func TestCheckForUpdateOffersTheInstallerWithoutDownloadingIt(t *testing.T) {
	suffix := updateSuffix(runtime.GOOS, runtime.GOARCH)
	if suffix == "" {
		t.Skip("no installer is published for this platform")
	}
	name := "GuildSync-Voice-Mute-9.9.9" + suffix
	assetURL := updateAssetBase + "v9.9.9/" + name
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"tag_name":"v9.9.9","html_url":%q,"assets":[{"name":%q,"browser_download_url":%q}]}`,
			updateReleaseURL+"/tag/v9.9.9", name, assetURL)
	}))
	defer server.Close()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	if err := os.Mkdir(filepath.Join(home, "Downloads"), 0o755); err != nil {
		t.Fatal(err)
	}
	opened := ""
	app := &App{updateURL: server.URL, openPath: func(path string) error { opened = path; return nil }}
	info, err := app.CheckForUpdate()
	if err != nil {
		t.Fatal(err)
	}
	if !info.Available || info.Version != "9.9.9" || info.AssetURL != assetURL {
		t.Fatalf("%+v", info)
	}
	if info.Message != "Version 9.9.9 is available." {
		t.Fatalf("message = %q", info.Message)
	}
	// Detecting an update must never start a download or open anything.
	if opened != "" {
		t.Fatalf("the check opened %q", opened)
	}
	if entries, err := os.ReadDir(filepath.Join(home, "Downloads")); err != nil || len(entries) != 0 {
		t.Fatalf("the check left files behind: %v, %v", entries, err)
	}
}

func TestDownloadUpdateFetchesTheInstallerForThisPlatform(t *testing.T) {
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
	app := &App{
		updateURL: server.URL + "/latest",
		assetBase: server.URL + "/download/",
		openPath:  func(path string) error { opened = path; return nil },
	}
	info, err := app.CheckForUpdate()
	if err != nil {
		t.Fatal(err)
	}
	if !info.Available || info.AssetURL != server.URL+"/download/"+name {
		t.Fatalf("%+v", info)
	}
	if opened != "" {
		t.Fatalf("the check downloaded and opened %q before being asked", opened)
	}
	message, err := app.DownloadUpdate()
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(opened) != name {
		t.Fatalf("opened %q, want %q", opened, name)
	}
	if contents, err := os.ReadFile(opened); err != nil || !bytes.Equal(contents, payload) {
		t.Fatalf("downloaded %q, %v", contents, err)
	}
	// Opening the file is all the UI does next, so on Linux the download has to
	// have left it runnable.
	if runtime.GOOS == "linux" {
		info, err := os.Stat(opened)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm()&0o111 == 0 {
			t.Fatalf("downloaded %q with mode %v, which cannot be run", opened, info.Mode().Perm())
		}
	}
	if !strings.Contains(message, name) || !strings.Contains(message, "opened it") {
		t.Fatalf("message does not report the download: %q", message)
	}
}

func TestCheckForUpdateSaysWhenTheReleaseHasNoInstaller(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"tag_name":"v9.9.9","html_url":"` + updateReleaseURL + `/tag/v9.9.9","assets":[]}`))
	}))
	defer server.Close()
	app := &App{updateURL: server.URL}
	info, err := app.CheckForUpdate()
	if err != nil {
		t.Fatal(err)
	}
	if !info.Available || info.AssetURL != "" || !strings.Contains(info.Message, "no installer for this platform") {
		t.Fatalf("%+v", info)
	}
	message, err := app.DownloadUpdate()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(message, "no installer for this platform") {
		t.Fatalf("download message = %q", message)
	}
}

func TestReleasePageIsValidated(t *testing.T) {
	if usableReleasePage("file:///etc/passwd") {
		t.Fatal("accepted a non-release URL")
	}
	if !usableReleasePage(updateReleaseURL) || !usableReleasePage(updateReleaseURL+"/tag/v1.0.0") {
		t.Fatal("rejected a release URL")
	}
	// The version the UI shows links straight at this platform's installer, so
	// that URL has to survive the same allow-list the release page does.
	if !usableReleasePage(updateAssetBase + "v9.9.9/GuildSync-Voice-Mute-9.9.9-Linux-x86_64.AppImage") {
		t.Fatal("rejected an installer URL")
	}
	if !strings.HasPrefix(updateAPIURL, "https://api.github.com/repos/"+updateRepo+"/") {
		t.Fatalf("update endpoint left the repository: %s", updateAPIURL)
	}
	if err := (&App{}).OpenReleasePage("https://example.com/evil.exe"); err == nil {
		t.Fatal("opened a foreign URL")
	}
	if err := (&App{}).OpenReleasePage(updateReleaseURL); err == nil {
		t.Fatal("missing application reported success")
	}
}
