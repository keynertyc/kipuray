package browser

import (
	"path/filepath"
	"slices"
	"testing"
)

func TestBrowserDiscoveryIncludesPlatformInstallations(t *testing.T) {
	roots := map[string]string{"LOCALAPPDATA": filepath.Join("user", "local"), "ProgramFiles": filepath.Join("system", "programs"), "ProgramFiles(x86)": filepath.Join("system", "programs-x86")}
	getenv := func(key string) string { return roots[key] }
	windows := chromeCandidates("windows", getenv, "")
	for _, root := range roots {
		expected := filepath.Join(root, "Google", "Chrome", "Application", "chrome.exe")
		if !slices.Contains(windows, expected) {
			t.Errorf("missing Windows browser installation %q", expected)
		}
	}
	home := filepath.Join("users", "developer")
	mac := chromeCandidates("darwin", getenv, home)
	expected := filepath.Join(home, "Applications", "Chromium.app", "Contents", "MacOS", "Chromium")
	if !slices.Contains(mac, expected) {
		t.Errorf("missing per-user macOS installation %q", expected)
	}
	if !slices.Contains(chromeCandidates("linux", getenv, ""), "google-chrome-stable") {
		t.Fatal("missing Chrome stable executable")
	}
}
