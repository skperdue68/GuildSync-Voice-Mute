package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	updateRepo       = "skperdue68/GuildSync-Voice-Mute"
	updateAPIURL     = "https://api.github.com/repos/" + updateRepo + "/releases/latest"
	updatePageURL    = "https://github.com/" + updateRepo + "/releases/latest"
	updatePagePrefix = "https://github.com/" + updateRepo + "/releases"
)

// UpdateInfo is the result of comparing the running build with the newest
// published release. Releases are unsigned zips, so the app opens the release
// page instead of downloading and running an installer itself.
type UpdateInfo struct {
	Current   string `json:"current"`
	Latest    string `json:"latest"`
	Available bool   `json:"available"`
	PageURL   string `json:"page_url"`
	Notes     string `json:"notes"`
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

func usableReleasePage(url string) bool {
	return url == updatePageURL || strings.HasPrefix(url, updatePagePrefix+"/") || url == updatePagePrefix
}

// CheckForUpdate asks GitHub for the newest published release. It never downloads
// or executes anything; the UI offers the release page instead.
func (a *App) CheckForUpdate() (UpdateInfo, error) {
	info := UpdateInfo{Current: productVersion(), PageURL: updatePageURL}
	endpoint := a.updateURL
	if endpoint == "" {
		endpoint = updateAPIURL
	}
	request, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return info, err
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("User-Agent", "GuildSync-Voice-Mute/"+info.Current)
	client := &http.Client{Timeout: 10 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		return info, fmt.Errorf("check for updates: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return info, fmt.Errorf("check for updates: %s", response.Status)
	}
	var release struct {
		TagName    string `json:"tag_name"`
		Body       string `json:"body"`
		HTMLURL    string `json:"html_url"`
		Draft      bool   `json:"draft"`
		Prerelease bool   `json:"prerelease"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&release); err != nil {
		return info, fmt.Errorf("check for updates: %w", err)
	}
	latest := strings.TrimPrefix(strings.TrimSpace(release.TagName), "v")
	if latest == "" {
		info.Message = "No published release was found."
		return info, nil
	}
	info.Latest = latest
	if release.Draft || release.Prerelease {
		info.Message = "The newest release is a pre-release; staying on " + info.Current + "."
		return info, nil
	}
	if compareVersions(latest, info.Current) > 0 {
		info.Available = true
		if usableReleasePage(release.HTMLURL) {
			info.PageURL = release.HTMLURL
		}
		info.Notes = strings.TrimSpace(release.Body)
		if len(info.Notes) > 1200 {
			info.Notes = info.Notes[:1200] + "…"
		}
		info.Message = "Version " + latest + " is available."
		return info, nil
	}
	info.Message = "You are running the latest release."
	return info, nil
}

// OpenReleasePage opens a GuildSync release page in the system browser, refusing
// any other URL so a crafted release payload cannot send the browser elsewhere.
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
