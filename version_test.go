package main

import (
	"strings"
	"testing"
)

func TestVersionMatchesEmbeddedProductMetadata(t *testing.T) {
	if productVersion() == "unknown" || productVersion() == "" {
		t.Fatal("missing embedded version")
	}
	if strings.ContainsAny(productVersion(), "#\"' ") {
		t.Fatalf("version leaked config syntax: %q", productVersion())
	}
	if (&App{}).GetAppVersion() != productVersion() {
		t.Fatal("window version differs")
	}
}
