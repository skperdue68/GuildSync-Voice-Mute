package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	updateRepo       = "skperdue68/GuildSync-Voice-Mute"
	updateAPIURL     = "https://api.github.com/repos/" + updateRepo + "/releases/latest"
	updateReleaseURL = "https://github.com/" + updateRepo + "/releases"
	updateAssetBase  = updateReleaseURL + "/download/"
	// A release installer is a few megabytes. Anything near this cap is not one,
	// so the download stops before it can fill the disk.
	maxUpdateBytes = 512 << 20
)

// UpdateInfo is what the UI needs from a check: whether a newer release exists,
// where to read about it and a sentence to show. A check that finds a newer
// release also downloads that platform's installer and opens it, so the message
// says what was downloaded and what to do next.
type UpdateInfo struct {
	Available bool   `json:"available"`
	PageURL   string `json:"page_url"`
	Message   string `json:"message"`
}

// versionParts compares dotted versions, ignoring a leading "v" and any
// pre-release or build suffix.
func versionParts(value string) ([3]int, bool) {
	parts := [3]int{}
	trimmed := strings.TrimPrefix(strings.TrimSpace(value), "v")
	if cut := strings.IndexAny(trimmed, "-+"); cut >= 0 {
		trimmed = trimmed[:cut]
	}
	if trimmed == "" {
		return parts, false
	}
	fields := strings.Split(trimmed, ".")
	if len(fields) > 3 {
		fields = fields[:3]
	}
	for index, field := range fields {
		number, err := strconv.Atoi(field)
		if err != nil {
			return parts, false
		}
		parts[index] = number
	}
	return parts, true
}

// compareVersions returns 1 when a is newer, -1 when b is newer and 0 when equal.
func compareVersions(a, b string) int {
	left, leftOK := versionParts(a)
	right, rightOK := versionParts(b)
	if !leftOK || !rightOK {
		return 0
	}
	for index := range left {
		switch {
		case left[index] > right[index]:
			return 1
		case left[index] < right[index]:
			return -1
		}
	}
	return 0
}

// usableReleasePage keeps a crafted payload from sending the browser anywhere
// outside this repository's release pages.
func usableReleasePage(url string) bool {
	return url == updateReleaseURL || strings.HasPrefix(url, updateReleaseURL+"/")
}

// updateAsset is one file attached to a release, as far as the updater needs it.
type updateAsset struct {
	Name   string
	URL    string
	Digest string
}

// updateSuffix names the installer each platform ships in its release.
func updateSuffix(goos string) string {
	switch goos {
	case "windows":
		return "-Windows-Setup.exe"
	case "darwin":
		return "-macOS.dmg"
	case "linux":
		return "-Linux.deb"
	}
	return ""
}

// pickUpdateAsset chooses the installer for one platform. The URL check keeps a
// crafted release payload from making the app download from anywhere else.
func pickUpdateAsset(assets []updateAsset, goos, base string) (updateAsset, bool) {
	suffix := updateSuffix(goos)
	if suffix == "" {
		return updateAsset{}, false
	}
	for _, asset := range assets {
		if strings.HasSuffix(asset.Name, suffix) && strings.HasPrefix(asset.URL, base) {
			return asset, true
		}
	}
	return updateAsset{}, false
}

// digestMatches checks GitHub's published "sha256:<hex>" digest against the
// bytes that were written. An empty digest means the release did not publish
// one, which is not a failure.
func digestMatches(digest, actual string) bool {
	algorithm, value, found := strings.Cut(digest, ":")
	return !found || strings.EqualFold(algorithm, "sha256") && strings.EqualFold(value, actual)
}

// downloadDir prefers the desktop's downloads folder and falls back to the
// temporary directory, which the message names either way.
func downloadDir() string {
	if home, err := os.UserHomeDir(); err == nil {
		downloads := filepath.Join(home, "Downloads")
		if info, err := os.Stat(downloads); err == nil && info.IsDir() {
			return downloads
		}
	}
	return os.TempDir()
}

// fetchUpdateAsset downloads one release asset into directory and returns its
// path. The bytes are written to a .part file and only renamed once the
// checksum matches, so a truncated or corrupted download is never left looking
// like a usable installer.
func fetchUpdateAsset(asset updateAsset, directory string) (string, error) {
	request, err := http.NewRequest(http.MethodGet, asset.URL, nil)
	if err != nil {
		return "", fmt.Errorf("building the request: %w", err)
	}
	request.Header.Set("User-Agent", "GuildSync-Voice-Mute/"+productVersion())
	response, err := (&http.Client{Timeout: 5 * time.Minute}).Do(request)
	if err != nil {
		return "", fmt.Errorf("talking to GitHub: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status %s", response.Status)
	}
	path := filepath.Join(directory, filepath.Base(asset.Name))
	partial := path + ".part"
	file, err := os.Create(partial)
	if err != nil {
		return "", fmt.Errorf("saving the file: %w", err)
	}
	hash := sha256.New()
	written, err := io.Copy(io.MultiWriter(file, hash), io.LimitReader(response.Body, maxUpdateBytes+1))
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}
	if err == nil && written > maxUpdateBytes {
		err = fmt.Errorf("the file is larger than %d MB", maxUpdateBytes>>20)
	}
	if err == nil && !digestMatches(asset.Digest, hex.EncodeToString(hash.Sum(nil))) {
		err = errors.New("the checksum does not match the published digest")
	}
	if err == nil {
		// Windows refuses to rename onto an existing file, so an earlier
		// download of the same version is replaced first.
		err = os.Rename(partial, path)
		if err != nil && os.Remove(path) == nil {
			err = os.Rename(partial, path)
		}
	}
	if err != nil {
		os.Remove(partial)
		return "", fmt.Errorf("saving the file: %w", err)
	}
	return path, nil
}

// openDownloaded hands the downloaded installer to the desktop so the user
// finishes installing it themselves.
func openDownloaded(path string) error {
	var command *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		command = exec.Command("rundll32", "url.dll,FileProtocolHandler", path)
	case "darwin":
		command = exec.Command("open", path)
	default:
		command = exec.Command("xdg-open", path)
	}
	if err := command.Start(); err != nil {
		return err
	}
	go func() { _ = command.Wait() }()
	return nil
}

func (a *App) openInstaller(path string) error {
	if a.openPath != nil {
		return a.openPath(path)
	}
	return openDownloaded(path)
}

// downloadUpdate fetches the installer for this platform from a newer release
// and returns the sentence the UI shows. A failed download is reported in that
// sentence rather than as an error, because the release page is then the way
// forward and the caller keeps it.
func (a *App) downloadUpdate(version string, assets []updateAsset) string {
	available := "Version " + version + " is available."
	base := a.assetBase
	if base == "" {
		base = updateAssetBase
	}
	asset, ok := pickUpdateAsset(assets, runtime.GOOS, base)
	if !ok {
		return available + " That release has no installer for this platform; open the download page instead."
	}
	path, err := fetchUpdateAsset(asset, downloadDir())
	if err != nil {
		return available + " The download could not be completed (" + err.Error() + "). Open the download page instead."
	}
	if err := a.openInstaller(path); err != nil {
		return available + " Downloaded " + asset.Name + " to " + path + "; open it to install, then start the new version."
	}
	return available + " Downloaded " + asset.Name + " to " + path + " and opened it. Quit the companion before installing, then start the new version."
}

// CheckForUpdate asks GitHub for the newest published release. The latest
// endpoint already skips drafts and pre-releases, so a pre-release is never
// offered. When the release is newer, its installer for this platform is
// downloaded and opened.
func (a *App) CheckForUpdate() (UpdateInfo, error) {
	info := UpdateInfo{PageURL: updateReleaseURL}
	endpoint := a.updateURL
	if endpoint == "" {
		endpoint = updateAPIURL
	}
	request, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return info, err
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("User-Agent", "GuildSync-Voice-Mute/"+productVersion())
	response, err := (&http.Client{Timeout: 10 * time.Second}).Do(request)
	if err != nil {
		return info, fmt.Errorf("check for updates: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return info, fmt.Errorf("check for updates: %s", response.Status)
	}
	var release struct {
		TagName string `json:"tag_name"`
		HTMLURL string `json:"html_url"`
		Assets  []struct {
			Name   string `json:"name"`
			URL    string `json:"browser_download_url"`
			Digest string `json:"digest"`
		} `json:"assets"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&release); err != nil {
		return info, fmt.Errorf("check for updates: %w", err)
	}
	latest := strings.TrimPrefix(strings.TrimSpace(release.TagName), "v")
	switch {
	case latest == "":
		info.Message = "No published release was found."
	case compareVersions(latest, productVersion()) > 0:
		info.Available = true
		if usableReleasePage(release.HTMLURL) {
			info.PageURL = release.HTMLURL
		}
		assets := make([]updateAsset, 0, len(release.Assets))
		for _, asset := range release.Assets {
			assets = append(assets, updateAsset{Name: asset.Name, URL: asset.URL, Digest: asset.Digest})
		}
		info.Message = a.downloadUpdate(latest, assets)
	default:
		info.Message = "You are running the latest release."
	}
	return info, nil
}

// OpenReleasePage opens a GuildSync release page in the system browser, refusing
// any other URL.
func (a *App) OpenReleasePage(url string) error {
	if !usableReleasePage(url) {
		return fmt.Errorf("only GuildSync release pages can be opened")
	}
	a.mu.Lock()
	app := a.app
	a.mu.Unlock()
	if app == nil {
		return fmt.Errorf("the application is not running")
	}
	return app.Browser.OpenURL(url)
}
