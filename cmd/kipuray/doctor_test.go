package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDoctorHonorsExplicitBrowserPath(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing-chrome")
	var output bytes.Buffer
	code, err := execute(context.Background(), []string{"doctor", "--chrome", missing}, &output)
	if code != 2 || err == nil || !strings.Contains(err.Error(), filepath.Base(missing)) {
		t.Fatalf("doctor ignored explicit path: code=%d err=%v output=%q", code, err, output.String())
	}
}

func TestMain(m *testing.M) {
	if os.Getenv("KIPURAY_TEST_BROWSER_PROCESS") == "1" && len(os.Args) == 2 && os.Args[1] == "--version" {
		fmt.Println("Kipuray synthetic browser 1.0")
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func TestDoctorReportsSelectedBrowserVersion(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("KIPURAY_TEST_BROWSER_PROCESS", "1")
	var output bytes.Buffer
	code, err := execute(context.Background(), []string{"doctor", "--chrome", executable}, &output)
	if code != 0 || err != nil || !strings.Contains(output.String(), "Kipuray synthetic browser 1.0") {
		t.Fatalf("selected browser was not used: code=%d err=%v output=%q", code, err, output.String())
	}
}

func TestDoctorHelpDoesNotStartBrowser(t *testing.T) {
	var output bytes.Buffer
	code, err := execute(context.Background(), []string{"doctor", "--help"}, &output)
	if code != 0 || err != nil || !strings.Contains(output.String(), "chrome") {
		t.Fatalf("expected doctor help: code=%d err=%v output=%q", code, err, output.String())
	}
}
