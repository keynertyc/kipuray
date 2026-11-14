package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/keynertyc/kipuray/internal/browser"
	"github.com/keynertyc/kipuray/internal/casefile"
	"github.com/keynertyc/kipuray/internal/demo"
	"github.com/keynertyc/kipuray/internal/httpapi"
	"github.com/keynertyc/kipuray/internal/report"
	"github.com/keynertyc/kipuray/internal/store"
)

var version = "dev"

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	code, err := execute(ctx, os.Args[1:], os.Stdout)
	if err != nil {
		fmt.Fprintln(os.Stderr, "kipuray:", err)
	}
	os.Exit(code)
}
func execute(ctx context.Context, args []string, out io.Writer) (int, error) {
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" {
		fmt.Fprintln(out, `Kipuray — local bug replay and fix verification

Usage:
  kipuray dev [options] <local-url>       Record a local application
  kipuray open [options] <case.kpr>       Import and view a portable case
  kipuray run [options] <case.kpr>        Execute against --target <local-url>
  kipuray compare [options] <case.kpr>    Compare --before and --after URLs
  kipuray serve [options]                Open the workspace or sharing server
  kipuray demo [options]                 Start broken/fixed checkout demos
  kipuray doctor                        Check the installed browser
  kipuray version                       Print version

Options can appear before or after the case/URL. Use <command> --help for details.
Run/compare exit codes: 0 passed/verified, 1 failed/not reproduced, 2 incomplete.`)
		return 0, nil
	}
	switch args[0] {
	case "version":
		fmt.Fprintln(out, version)
		return 0, nil
	case "doctor":
		return doctor(ctx, args[1:], out)
	case "run", "compare":
		return runCase(ctx, args[0], args[1:], out)
	case "serve", "dev", "open", "demo":
		return serve(ctx, args[0], args[1:], out)
	default:
		return 2, fmt.Errorf("unknown command %q; use kipuray help", args[0])
	}
}

// reorder permits the natural `run case.kpr --target URL` form with Go's flag package.
func reorder(args []string) []string {
	flags := []string{}
	positionals := []string{}
	bools := map[string]bool{"--no-open": true, "--headless": true, "--capture-json": true}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if len(arg) > 0 && arg[0] == '-' {
			flags = append(flags, arg)
			if !bools[arg] && !containsEqual(arg) && arg != "--help" && arg != "-h" && i+1 < len(args) {
				i++
				flags = append(flags, args[i])
			}
		} else {
			positionals = append(positionals, arg)
		}
	}
	return append(flags, positionals...)
}
func containsEqual(s string) bool {
	for _, c := range s {
		if c == '=' {
			return true
		}
	}
	return false
}
func runCase(ctx context.Context, command string, args []string, out io.Writer) (int, error) {
	f := flag.NewFlagSet(command, flag.ContinueOnError)
	target := f.String("target", "", "local target URL")
	before := f.String("before", "", "previous version URL")
	after := f.String("after", "", "corrected version URL")
	variant := f.String("variant", "", "variant ID")
	format := f.String("format", "json", "json, junit, or html")
	output := f.String("output", "", "write report to a file")
	chrome := f.String("chrome", "", "browser executable path")
	if err := f.Parse(reorder(args)); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0, nil
		}
		return 2, err
	}
	if f.NArg() != 1 {
		return 2, errors.New("provide one .kpr file")
	}
	if *format != "json" && *format != "junit" && *format != "html" {
		return 2, errors.New("format must be json, junit, or html")
	}
	file, err := os.Open(f.Arg(0))
	if err != nil {
		return 2, err
	}
	if err := checkReportOutput(file, *output); err != nil {
		file.Close()
		return 2, err
	}
	c, err := casefile.Read(file)
	file.Close()
	if err != nil {
		return 2, err
	}
	opts := browser.Options{Headless: true, ExecPath: *chrome}
	var result any
	status := "incomplete"
	if command == "compare" {
		if *before == "" || *after == "" {
			return 2, errors.New("compare requires --before and --after")
		}
		b := browser.Run(ctx, c, *before, *variant, opts)
		a := browser.Run(ctx, c, *after, *variant, opts)
		comparison := browser.Compare(b, a)
		result = comparison
		status = comparison.Status
	} else {
		if *target == "" {
			return 2, errors.New("run requires --target")
		}
		r := browser.Run(ctx, c, *target, *variant, opts)
		result = r
		status = r.Status
	}
	writer := out
	if *output != "" {
		file, err := os.Create(*output)
		if err != nil {
			return 2, err
		}
		defer file.Close()
		writer = file
	}
	if err := report.Write(writer, *format, result); err != nil {
		return 2, err
	}
	if status == "incomplete" {
		return 2, nil
	}
	if status != "passed" && status != "verified" {
		return 1, nil
	}
	return 0, nil
}
func serve(ctx context.Context, command string, args []string, out io.Writer) (int, error) {
	f := flag.NewFlagSet(command, flag.ContinueOnError)
	listen := f.String("listen", "127.0.0.1:4317", "listen address")
	data := f.String("data-dir", defaultData(), "workspace directory")
	noOpen := f.Bool("no-open", false, "do not open the workspace browser")
	headless := f.Bool("headless", false, "record without a visible browser")
	captureJSON := f.Bool("capture-json", false, "capture synthetic JSON payloads (review before sharing)")
	chrome := f.String("chrome", "", "browser executable path")
	if err := f.Parse(reorder(args)); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0, nil
		}
		return 2, err
	}
	host, _, err := net.SplitHostPort(*listen)
	if err != nil {
		return 2, err
	}
	ip := net.ParseIP(host)
	local := host == "localhost" || (ip != nil && ip.IsLoopback())
	token := os.Getenv("KIPURAY_TOKEN")
	if !local && len(token) < 32 {
		return 2, errors.New("remote serving requires KIPURAY_TOKEN with at least 32 characters")
	}
	if command != "serve" && !local {
		return 2, errors.New("dev, open, and demo require a loopback listen address")
	}
	db, err := store.Open(*data)
	if err != nil {
		return 2, err
	}
	defer db.Close()
	path := "/"
	if command == "open" {
		if f.NArg() != 1 {
			return 2, errors.New("provide one .kpr file")
		}
		file, err := os.Open(f.Arg(0))
		if err != nil {
			return 2, err
		}
		c, err := casefile.Read(file)
		file.Close()
		if err != nil {
			return 2, err
		}
		c.ID = casefile.NewID()
		if err := db.Put(ctx, c); err != nil {
			return 2, err
		}
		path = "/?case=" + c.ID
	}
	var extras []*http.Server
	if command == "demo" {
		for i, addr := range []string{"127.0.0.1:3000", "127.0.0.1:3001"} {
			l, err := net.Listen("tcp", addr)
			if err != nil {
				for _, s := range extras {
					_ = s.Close()
				}
				return 2, fmt.Errorf("demo address %s: %w", addr, err)
			}
			s := &http.Server{Handler: demo.Handler(i == 1), ReadHeaderTimeout: 5 * time.Second}
			extras = append(extras, s)
			go func() { _ = s.Serve(l) }()
		}
		defer func() {
			for _, s := range extras {
				_ = s.Close()
			}
		}()
		fmt.Fprintln(out, "Demo before: http://127.0.0.1:3000\nDemo after:  http://127.0.0.1:3001")
		fmt.Fprintln(out, "Recording the synthetic demo in Chrome…")
		sample, err := browser.RecordDemo(ctx, "http://127.0.0.1:3000", browser.Options{ExecPath: *chrome})
		if err != nil {
			return 2, fmt.Errorf("record demo: %w", err)
		}
		if err := db.Put(ctx, sample); err != nil {
			return 2, err
		}
		path = "/?case=" + sample.ID
	}
	if command == "dev" {
		if f.NArg() != 1 {
			return 2, errors.New("provide a local URL")
		}
		if err := browser.CheckTarget(f.Arg(0)); err != nil {
			return 2, err
		}
		path = "/?target=" + url.QueryEscape(f.Arg(0))
		if *captureJSON {
			path += "&captureJSON=true"
		}
	}
	listener, err := net.Listen("tcp", *listen)
	if err != nil {
		return 2, err
	}
	allowedHost := ""
	if local {
		allowedHost = listener.Addr().String()
	}
	api := httpapi.New(db, httpapi.Config{Local: local, OwnerToken: token, Host: allowedHost, Browser: browser.Options{Headless: *headless, ExecPath: *chrome}, Context: ctx})
	defer api.Close()
	server := &http.Server{Handler: api, ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 1 << 20}
	address := "http://" + listener.Addr().String() + path
	fmt.Fprintln(out, "Workspace:", address)
	if !*noOpen && local {
		openBrowser(address)
	}
	done := make(chan error, 1)
	go func() { done <- server.Serve(listener) }()
	select {
	case <-ctx.Done():
		shutdown, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		_ = server.Shutdown(shutdown)
		return 0, nil
	case err := <-done:
		if errors.Is(err, http.ErrServerClosed) {
			return 0, nil
		}
		return 2, err
	}
}
func defaultData() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "./data"
	}
	return filepath.Join(dir, "kipuray")
}
func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	_ = cmd.Run()
}
