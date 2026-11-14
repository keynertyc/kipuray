package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/keynertyc/kipuray/internal/casefile"
)

func TestReportCannotOverwriteItsInputCase(t *testing.T) {
	for _, command := range []string{"run", "compare"} {
		t.Run(command, func(t *testing.T) {
			dir := t.TempDir()
			input := filepath.Join(dir, "case.kpr")
			var archive bytes.Buffer
			if err := casefile.Write(&archive, casefile.New("http://localhost:3000")); err != nil {
				t.Fatal(err)
			}
			original := append([]byte(nil), archive.Bytes()...)
			if err := os.WriteFile(input, original, 0600); err != nil {
				t.Fatal(err)
			}
			for _, alias := range []string{"same-path", "hard-link", "symbolic-link"} {
				t.Run(alias, func(t *testing.T) {
					if err := os.WriteFile(input, original, 0600); err != nil {
						t.Fatal(err)
					}
					output := input
					if alias != "same-path" {
						output = filepath.Join(dir, alias+".kpr")
						var err error
						if alias == "hard-link" {
							err = os.Link(input, output)
						} else {
							err = os.Symlink(input, output)
						}
						if err != nil {
							t.Skipf("filesystem does not support %s: %v", alias, err)
						}
					}
					args := []string{command, input, "--output", output}
					if command == "run" {
						args = append(args, "--target", "http://localhost:3000")
					} else {
						args = append(args, "--before", "http://localhost:3000", "--after", "http://localhost:3001")
					}
					var stdout bytes.Buffer
					code, err := execute(context.Background(), args, &stdout)
					if code != 2 || err == nil || !strings.Contains(err.Error(), "input case") {
						t.Errorf("expected input protection: code=%d error=%v", code, err)
					}
					data, readErr := os.ReadFile(input)
					if readErr != nil || !bytes.Equal(data, original) {
						t.Errorf("input case was modified: %v", readErr)
					}
				})
			}
		})
	}
}

func TestReportCanBeWrittenToAnotherFile(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "case.kpr")
	output := filepath.Join(dir, "result.json")
	var archive bytes.Buffer
	if err := casefile.Write(&archive, casefile.New("http://localhost:3000")); err != nil {
		t.Fatal(err)
	}
	original := append([]byte(nil), archive.Bytes()...)
	if err := os.WriteFile(input, original, 0600); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	code, err := execute(context.Background(), []string{"run", input, "--target", "http://localhost:3000", "--output", output}, &stdout)
	// No expectations: a valid incomplete report, with no browser required.
	if code != 2 || err != nil {
		t.Fatalf("code=%d error=%v", code, err)
	}
	data, err := os.ReadFile(output)
	if err != nil || !bytes.Contains(data, []byte(`"status": "incomplete"`)) {
		t.Fatalf("missing report: %s %v", data, err)
	}
	got, err := os.ReadFile(input)
	if err != nil || !bytes.Equal(got, original) {
		t.Fatal("input archive changed")
	}
}
