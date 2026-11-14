package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os/exec"
	"time"

	"github.com/keynertyc/kipuray/internal/browser"
)

func doctor(ctx context.Context, args []string, out io.Writer) (int, error) {
	flags := flag.NewFlagSet("doctor", flag.ContinueOnError)
	flags.SetOutput(out)
	explicit := flags.String("chrome", "", "browser executable path")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0, nil
		}
		return 2, err
	}
	if flags.NArg() != 0 {
		return 2, errors.New("doctor does not accept positional arguments")
	}
	path := *explicit
	if path == "" {
		path = browser.ChromePath()
	}
	if path == "" {
		return 2, errors.New("Chrome/Chromium not found; install it or pass doctor --chrome /path/to/browser")
	}
	resolved, err := exec.LookPath(path)
	if err != nil {
		return 2, fmt.Errorf("browser %q: %w", path, err)
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	output, err := exec.CommandContext(ctx, resolved, "--version").CombinedOutput()
	if err != nil {
		return 2, fmt.Errorf("browser %q could not report its version: %w", resolved, err)
	}
	if _, err := fmt.Fprintf(out, "Browser: %s\n%s", resolved, output); err != nil {
		return 2, err
	}
	return 0, nil
}
