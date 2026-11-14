package browser

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/keynertyc/kipuray/internal/casefile"
	"github.com/keynertyc/kipuray/internal/demo"
)

func TestPrivacy(t *testing.T) {
	body, changed := ScrubJSON(`{"token":"secret-fixture","nested":{"password":"secret-password"},"discount":null}`)
	if !changed || strings.Contains(body, "secret-") {
		t.Fatal(body)
	}
	raw, _ := safeURL("http://localhost:3000/api?token=secret-url&code=FIELD10")
	if strings.Contains(raw, "secret-url") {
		t.Fatal(raw)
	}
}

func TestPrivacyPreservesJSONNumberPrecision(t *testing.T) {
	body, redacted := ScrubJSON(`{"id":9007199254740993,"token":"sensitive"}`)
	if !redacted || !strings.Contains(body, "9007199254740993") || strings.Contains(body, "sensitive") {
		t.Fatalf("unexpected redacted payload: %s", body)
	}
	if body, _ := ScrubJSON(`{"id":1} {"id":2}`); body != "" {
		t.Fatal("accepted more than one JSON value")
	}
}

func TestConcurrentRequestsStayNonExecutable(t *testing.T) {
	if os.Getenv("KIPURAY_BROWSER_TEST") != "1" {
		t.Skip("requires Chrome")
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api" {
			time.Sleep(100 * time.Millisecond)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"ok":true}`))
			return
		}
		_, _ = w.Write([]byte(`<!doctype html><button id="go" onclick="Promise.all([fetch('/api'),fetch('/api')])">Go</button>`))
	}))
	defer server.Close()
	capture, err := StartCapture(context.Background(), server.URL, CaptureOptions{Options: Options{Headless: true}, CaptureJSON: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := chromedp.Do(capture.ctx, chromedp.Click(chromedp.CSS("#go"))); err != nil {
		capture.Stop()
		t.Fatal(err)
	}
	until := time.Now().Add(3 * time.Second)
	for len(capture.Snapshot().Fixtures) < 2 && time.Now().Before(until) {
		time.Sleep(20 * time.Millisecond)
	}
	c := capture.Stop()
	if len(c.ExecutionProblems) == 0 {
		t.Fatal("ambiguous concurrent scenario accepted")
	}
}
func TestTargetScope(t *testing.T) {
	for _, s := range []string{"https://example.com", "file:///tmp/a", "http://127.0.0.1.evil.com", "http://user:pass@localhost:3000"} {
		if CheckTarget(s) == nil {
			t.Fatal("accepted", s)
		}
	}
	for _, s := range []string{"http://127.0.0.1:3000", "http://localhost:3000", "http://[::1]:3000"} {
		if err := CheckTarget(s); err != nil {
			t.Fatal(err)
		}
	}
}
func TestComparisonSemantics(t *testing.T) {
	for _, tc := range []struct{ before, after, want string }{{"failed", "passed", "verified"}, {"passed", "passed", "not-reproduced"}, {"failed", "failed", "still-failing"}, {"passed", "failed", "regression"}, {"incomplete", "passed", "incomplete"}} {
		if got := Compare(Result{Status: tc.before}, Result{Status: tc.after}); got.Status != tc.want {
			t.Fatal(got)
		}
	}
}

func TestBrowserCaptureReplayAndVerify(t *testing.T) {
	if os.Getenv("KIPURAY_BROWSER_TEST") != "1" {
		t.Skip("set KIPURAY_BROWSER_TEST=1 to run real Chromium integration")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	old := httptest.NewServer(demo.Handler(false))
	defer old.Close()
	fixed := httptest.NewServer(demo.Handler(true))
	defer fixed.Close()
	capture, err := StartCapture(ctx, old.URL+"?token=KIPURAY_SEEDED_SECRET", CaptureOptions{Options: Options{Headless: true}, CaptureJSON: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := chromedp.Do(capture.ctx, chromedp.WaitVisible(chromedp.CSS(`[data-testid="apply"]`)), chromedp.Click(chromedp.CSS(`[data-testid="apply"]`))); err != nil {
		capture.Stop()
		t.Fatal(err)
	}
	until := time.Now().Add(5 * time.Second)
	for len(capture.Snapshot().Fixtures) == 0 && time.Now().Before(until) {
		time.Sleep(30 * time.Millisecond)
	}
	time.Sleep(200 * time.Millisecond)
	c := capture.Stop()
	if data, _ := json.Marshal(c); strings.Contains(string(data), "KIPURAY_SEEDED_SECRET") {
		t.Fatal("credential escaped URL redaction")
	}
	c.Title = "Coupon null regression"
	c.Expectations = []casefile.Expectation{{Kind: "text", Selector: `[data-testid="message"]`, Value: "Coupon not valid"}}
	if len(c.Events) < 2 || len(c.Actions) == 0 || len(c.Fixtures) != 1 {
		b, _ := json.Marshal(c)
		t.Fatalf("incomplete capture: %s", b)
	}
	var archive bytes.Buffer
	if err := casefile.Write(&archive, c); err != nil {
		t.Fatal(err)
	}
	c, err = casefile.Read(&archive)
	if err != nil {
		t.Fatal(err)
	}
	before := Run(ctx, c, old.URL, "", Options{Headless: true})
	after := Run(ctx, c, fixed.URL, "", Options{Headless: true})
	comparison := Compare(before, after)
	if comparison.Status != "verified" {
		t.Fatalf("want verified, got before=%s after=%s problems=%v / %v steps=%v", before.Status, after.Status, before.Problems, after.Problems, after.Steps)
	}
	if got := Compare(after, after).Status; got != "not-reproduced" {
		t.Fatal(got)
	}
	c.Variants = []casefile.Variant{{ID: casefile.NewID(), Name: "Unavailable", FixtureID: c.Fixtures[0].ID, Body: `{"message":"try later"}`, Status: 500, DelayMS: 25}}
	c.Expectations[0].Value = "Service unavailable"
	variant := Run(ctx, c, fixed.URL, c.Variants[0].ID, Options{Headless: true})
	if variant.Status != "passed" {
		t.Fatalf("variant: %s %v", variant.Status, variant.Problems)
	}
	c.Fixtures = nil
	c.Variants = nil
	missing := Run(ctx, c, fixed.URL, "", Options{Headless: true})
	if missing.Status != "incomplete" {
		t.Fatalf("missing fixture should be incomplete: %s", missing.Status)
	}
}
