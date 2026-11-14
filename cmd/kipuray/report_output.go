package main

import (
	"errors"
	"fmt"
	"os"
)

// checkReportOutput protects the portable input even when the output reaches it
// through a symlink or hard link. Check before starting any browser work.
func checkReportOutput(input *os.File, output string) error {
	if output == "" {
		return nil
	}
	target, err := os.Stat(output)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("report output: %w", err)
	}
	source, err := input.Stat()
	if err != nil {
		return err
	}
	if os.SameFile(source, target) {
		return errors.New("report output cannot overwrite the input case; choose a different --output path")
	}
	if target.IsDir() {
		return errors.New("report output must be a file, not a directory")
	}
	return nil
}
