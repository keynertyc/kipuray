package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestCLIArguments(t *testing.T) {
	for _, args := range [][]string{{"help"}, {"run", "--help"}, {"compare", "--help"}, {"serve", "--help"}} {
		var out bytes.Buffer
		code, err := execute(context.Background(), args, &out)
		if code != 0 || err != nil {
			t.Fatalf("%v: %d %v", args, code, err)
		}
	}
	var out bytes.Buffer
	if code, err := execute(context.Background(), []string{"run", "does-not-exist.kpr", "--target", "http://localhost:3000"}, &out); code != 2 || err == nil || !strings.Contains(err.Error(), "does-not-exist") {
		t.Fatalf("%d %v", code, err)
	}
}
