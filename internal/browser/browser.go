// Package browser owns dedicated Chromium sessions. It never attaches to a
// developer's regular browser profile or executes commands from imported cases.
package browser

import (
	"context"
	_ "embed"
	"errors"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/chromedp/chromedp"
)

//go:embed recorder.js
var recorder string

type Options struct {
	Headless bool
	ExecPath string
}

func CheckTarget(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.User != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return errors.New("target must be a local HTTP or HTTPS URL")
	}
	host := u.Hostname()
	ip := net.ParseIP(host)
	if host != "localhost" && (ip == nil || !ip.IsLoopback()) {
		return errors.New("execution is restricted to localhost and loopback addresses")
	}
	return nil
}
func ChromePath() string {
	home, _ := os.UserHomeDir()
	for _, name := range chromeCandidates(runtime.GOOS, os.Getenv, home) {
		if p, err := exec.LookPath(name); err == nil {
			return p
		}
	}
	return ""
}

func chromeCandidates(platform string, getenv func(string) string, home string) []string {
	candidates := []string{"google-chrome", "google-chrome-stable", "chromium", "chromium-browser", "chrome", "chrome.exe", "msedge", "msedge.exe"}
	switch platform {
	case "darwin":
		locations := []string{"/Applications"}
		if home != "" {
			locations = append(locations, filepath.Join(home, "Applications"))
		}
		for _, dir := range locations {
			for _, app := range []struct{ bundle, binary string }{{"Google Chrome", "Google Chrome"}, {"Chromium", "Chromium"}, {"Microsoft Edge", "Microsoft Edge"}} {
				candidates = append(candidates, filepath.Join(dir, app.bundle+".app", "Contents", "MacOS", app.binary))
			}
		}
	case "windows":
		for _, key := range []string{"LOCALAPPDATA", "ProgramFiles", "ProgramFiles(x86)"} {
			if root := getenv(key); root != "" {
				for _, suffix := range [][]string{{"Google", "Chrome", "Application", "chrome.exe"}, {"Chromium", "Application", "chrome.exe"}, {"Microsoft", "Edge", "Application", "msedge.exe"}} {
					parts := append([]string{root}, suffix...)
					candidates = append(candidates, filepath.Join(parts...))
				}
			}
		}
	}
	return candidates
}
func launch(parent context.Context, o Options) (context.Context, context.CancelFunc, error) {
	options := append([]chromedp.ExecAllocatorOption{}, chromedp.DefaultExecAllocatorOptions[:]...)
	options = append(options, chromedp.Flag("headless", o.Headless), chromedp.Flag("disable-dev-shm-usage", true), chromedp.Flag("disable-background-networking", true), chromedp.WindowSize(1280, 800))
	path := o.ExecPath
	if path == "" {
		path = ChromePath()
	}
	if path != "" {
		options = append(options, chromedp.ExecPath(path))
	}
	alloc, stopAlloc := chromedp.NewExecAllocator(parent, options...)
	ctx, stop := chromedp.NewContext(alloc)
	cancel := func() { stop(); stopAlloc() }
	if err := chromedp.Do(ctx); err != nil {
		cancel()
		return nil, nil, err
	}
	return ctx, cancel, nil
}
