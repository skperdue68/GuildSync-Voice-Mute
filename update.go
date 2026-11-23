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
// the version on offer, the release page and — when that release published one
// for this machine — a direct link to this platform's installer. Checking never
// downloads anything; DownloadUpdate does that once the user asks.
type UpdateInfo struct {
	Available bool   `json:"available"`
	Version   string `json:"version"`
	PageURL   string `json:"page_url"`
	AssetURL  string `json:"asset_url"`
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

// linuxArchitecture names the AppImage for a Go architecture. Linux ships an
// AppImage so one file runs on any distribution, but it is still built for one
// architecture, and the generator only builds these two.
func linuxArchitecture(goarch string) string {
	switch goarch {
	case "amd64":
		return "x86_64"
	case "arm64":
		return "aarch64"
	}
	return ""
}

// updateSuffix names the file an update downloads for each platform. Every
// release publishes the installer next to a zip of that installer, and the zip
// is what an update fetches, so each name ends in ".zip". The Windows setup and
// the macOS disk image work on every architecture those releases target; a Linux
// AppImage does not, so its architecture is part of the name and is picked to
// match the machine.
func updateSuffix(goos, goarch string) string {
	switch goos {
	case "windows":
		return "-Windows-Setup.exe.zip"
	case "darwin":
		return "-macOS.dmg.zip"
	case "linux":
		if architecture := linuxArchitecture(goarch); architecture != "" {
			return "-Linux-" + architecture + ".AppImage.zip"
		}
	}
	return ""
}

// pickUpdateAsset chooses the installer for one platform and architecture. The
// URL check keeps a crafted release payload from making the app download from
// anywhere else.
func pickUpdateAsset(assets []updateAsset, goos, goarch, base string) (updateAsset, bool) {
	suffix := updateSuffix(goos, goarch)
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

// DownloadUpdate fetches the installer the newest release publishes for this
// platform, verifies its digest and hands it to the desktop. It looks the
// release up again rather than trusting an earlier check, so it shares no state
// with the check that fills the window. The returned sentence is what the UI
// shows; a refused download still returns one, because the release page is then
// the way forward.
func (a *App) DownloadUpdate() (string, error) {
	info, asset, err := a.lookupUpdate()
	if err != nil {
		return "", err
	}
	// Already current, or a release with no installer for this platform: the
	// check's own sentence says which, and the release page is the way forward.
	if !info.Available || asset == nil {
		return info.Message, nil
	}
	path, err := fetchUpdateAsset(*asset, downloadDir())
	if err != nil {
		return "The download could not be completed (" + err.Error() + "). Open the download page instead.", nil
	}
	if err := a.openInstaller(path); err != nil {
		return "Downloaded " + asset.Name + " to " + path + "; unzip it, run the installer inside, then start the new version.", nil
	}
	return "Downloaded " + asset.Name + " to " + path + " and opened it. Unzip it, quit the companion, run the installer inside, then start the new version.", nil
}

// lookupUpdate asks GitHub for the newest published release and describes it.
// The latest endpoint already skips drafts and pre-releases, so a pre-release is
// never offered.
func (a *App) lookupUpdate() (UpdateInfo, *updateAsset, error) {
	info := UpdateInfo{PageURL: updateReleaseURL}
	endpoint := a.updateURL
	if endpoint == "" {
		endpoint = updateAPIURL
	}
	request, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return info, nil, err
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("User-Agent", "GuildSync-Voice-Mute/"+productVersion())
	response, err := (&http.Client{Timeout: 10 * time.Second}).Do(request)
	if err != nil {
		return info, nil, fmt.Errorf("check for updates: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return info, nil, fmt.Errorf("check for updates: %s", response.Status)
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
		return info, nil, fmt.Errorf("check for updates: %w", err)
	}
	latest := strings.TrimPrefix(strings.TrimSpace(release.TagName), "v")
	switch {
	case latest == "":
		info.Message = "No published release was found."
	case compareVersions(latest, productVersion()) > 0:
		info.Available = true
		info.Version = latest
		if usableReleasePage(release.HTMLURL) {
			info.PageURL = release.HTMLURL
		}
		assets := make([]updateAsset, 0, len(release.Assets))
		for _, asset := range release.Assets {
			assets = append(assets, updateAsset{Name: asset.Name, URL: asset.URL, Digest: asset.Digest})
		}
		base := a.assetBase
		if base == "" {
			base = updateAssetBase
		}
		if asset, ok := pickUpdateAsset(assets, runtime.GOOS, runtime.GOARCH, base); ok {
			info.AssetURL = asset.URL
			info.Message = "Version " + latest + " is available."
			return info, &asset, nil
		}
		info.Message = "Version " + latest + " is available, but that release has no installer for this platform."
	default:
		info.Message = "You are running the latest release."
	}
	return info, nil, nil
}

// CheckForUpdate asks GitHub for the newest published release. It downloads
// nothing: the UI shows the version and links to this platform's installer, and
// DownloadUpdate fetches it when the user asks.
func (a *App) CheckForUpdate() (UpdateInfo, error) {
	info, _, err := a.lookupUpdate()
	return info, err
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
