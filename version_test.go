package main

import "testing"

func TestVersionMatchesEmbeddedProductMetadata(t *testing.T) {
	if productVersion() == "unknown" || productVersion() == "" {
		t.Fatal("missing embedded version")
	}
	if (&App{}).GetAppVersion() != productVersion() {
		t.Fatal("window version differs")
	}
	if (&App{}).GetPlatform() == "" {
		t.Fatal("missing permission platform")
	}
}
