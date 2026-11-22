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
	updateReleaseURL = "https://github.com/" + updateRepo + "/releases"
)

// UpdateInfo is what the UI needs from a check: whether a newer release exists,
// where to read about it and a sentence to show. Releases are unsigned archives,
// so the app opens the release page instead of downloading and running anything.
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

// CheckForUpdate asks GitHub for the newest published release. The latest
// endpoint already skips drafts and pre-releases, so a pre-release is never
// offered as an update.
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
		info.Message = "Version " + latest + " is available."
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
