package browser

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/keynertyc/kipuray/internal/casefile"
)

func TestRunWaitsForSlowFixturesBeforeCheckingExpectations(t *testing.T) {
	if os.Getenv("KIPURAY_BROWSER_TEST") != "1" {
		t.Skip("requires Chrome")
	}
	// The page becomes visible immediately. Its error appears only after the
	// recorded response resolves, so checking no-errors before the request settles
	// would incorrectly approve this case.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api" {
			t.Error("runner contacted the live API instead of using the fixture")
			w.WriteHeader(500)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<!doctype html><h1 id="status">Ready</h1><script>fetch('/api').then(r=>r.json()).then(()=>{throw new Error('failure after API response')})</script>`))
	}))
	defer server.Close()
	c := casefile.New(server.URL)
	c.Expectations = []casefile.Expectation{{Kind: "text", Selector: "#status", Value: "Ready"}, {Kind: "no-errors"}}
	c.Fixtures = []casefile.Fixture{{ID: casefile.NewID(), Method: "GET", URL: server.URL + "/api", Status: 200, ContentType: "application/json", Body: `{"ok":true}`, DelayMS: 350}}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	result := Run(ctx, c, server.URL, "", Options{Headless: true})
	if result.Status != "failed" {
		t.Fatalf("late browser error must fail: status=%s steps=%v problems=%v", result.Status, result.Steps, result.Problems)
	}
	if len(result.Steps) != 2 || result.Steps[1].Status != "failed" {
		t.Fatalf("no-errors did not observe delayed failure: %v", result.Steps)
	}
}

func TestRunAllowsSlowResponsesWithinItsDeadline(t *testing.T) {
	if os.Getenv("KIPURAY_BROWSER_TEST") != "1" {
		t.Skip("requires Chrome")
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<!doctype html><h1 id="status">Ready</h1><script>fetch('/api').then(r=>r.json())</script>`))
	}))
	defer server.Close()
	c := casefile.New(server.URL)
	c.Expectations = []casefile.Expectation{{Kind: "visible", Selector: "#status"}, {Kind: "no-errors"}}
	c.Fixtures = []casefile.Fixture{{ID: casefile.NewID(), Method: "GET", URL: server.URL + "/api", Status: 200, ContentType: "application/json", Body: `{"ok":true}`, DelayMS: 1500}}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	result := Run(ctx, c, server.URL, "", Options{Headless: true})
	if result.Status != "passed" {
		t.Fatalf("slow fixture should complete: status=%s problems=%v steps=%v", result.Status, result.Problems, result.Steps)
	}
}

func TestNoErrorsCoversLaterFunctionalChecks(t *testing.T) {
	if os.Getenv("KIPURAY_BROWSER_TEST") != "1" {
		t.Skip("requires Chrome")
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<!doctype html><h1 id="status">Waiting</h1><script>setTimeout(()=>fetch('/api').then(r=>r.json()).then(()=>{document.querySelector('#status').textContent='Loaded';throw new Error('failure during UI update')}),400)</script>`))
	}))
	defer server.Close()
	c := casefile.New(server.URL)
	c.Expectations = []casefile.Expectation{{Kind: "no-errors"}, {Kind: "text", Selector: "#status", Value: "Loaded"}}
	c.Fixtures = []casefile.Fixture{{ID: casefile.NewID(), Method: "GET", URL: server.URL + "/api", Status: 200, ContentType: "application/json", Body: `{"ok":true}`}}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	result := Run(ctx, c, server.URL, "", Options{Headless: true})
	if result.Status != "failed" || len(result.Steps) != 2 || result.Steps[0].Status != "failed" || result.Steps[1].Status != "passed" {
		t.Fatalf("error expectation did not cover the full check: status=%s steps=%v problems=%v", result.Status, result.Steps, result.Problems)
	}
}
