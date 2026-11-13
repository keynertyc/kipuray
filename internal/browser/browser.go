// Package browser owns dedicated Chromium sessions. It never attaches to a
// developer's regular browser profile or executes commands from imported cases.
package browser

import (
	"context"
	_ "embed"
	"errors"
	"net"
	"net/url"
	"os/exec"
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
	for _, name := range []string{"google-chrome", "chromium", "chromium-browser", "chrome"} {
		if p, err := exec.LookPath(name); err == nil {
			return p
		}
	}
	if runtime.GOOS == "darwin" {
		return "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"
	}
	return ""
}
func launch(parent context.Context, o Options) (context.Context, context.CancelFunc, error) {
	options := append([]chromedp.ExecAllocatorOption{}, chromedp.DefaultExecAllocatorOptions[:]...)
	options = append(options, chromedp.Flag("headless", o.Headless), chromedp.Flag("disable-dev-shm-usage", true), chromedp.Flag("disable-background-networking", true), chromedp.WindowSize(1280, 800))
	if o.ExecPath != "" {
		options = append(options, chromedp.ExecPath(o.ExecPath))
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
