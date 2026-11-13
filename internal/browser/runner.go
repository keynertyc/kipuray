package browser

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/cdproto/fetch"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/page"
	cdpruntime "github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
	"github.com/keynertyc/kipuray/internal/casefile"
)

type Step struct {
	Label  string `json:"label"`
	Status string `json:"status"`
	Detail string `json:"detail,omitempty"`
}
type Result struct {
	ID         string    `json:"id"`
	CaseID     string    `json:"caseId"`
	Target     string    `json:"target"`
	VariantID  string    `json:"variantId,omitempty"`
	Status     string    `json:"status"`
	StartedAt  time.Time `json:"startedAt"`
	DurationMS int64     `json:"durationMs"`
	Steps      []Step    `json:"steps"`
	Problems   []string  `json:"problems"`
	Screenshot string    `json:"screenshot,omitempty"`
}
type Comparison struct {
	Status string `json:"status"`
	Before Result `json:"before"`
	After  Result `json:"after"`
}

func Compare(before, after Result) Comparison {
	status := "incomplete"
	if before.Status != "incomplete" && after.Status != "incomplete" {
		switch {
		case before.Status == "failed" && after.Status == "passed":
			status = "verified"
		case before.Status == "passed" && after.Status == "passed":
			status = "not-reproduced"
		case before.Status == "passed" && after.Status == "failed":
			status = "regression"
		default:
			status = "still-failing"
		}
	}
	return Comparison{status, before, after}
}

type fixtureQueue struct {
	mu       sync.Mutex
	items    map[string][]casefile.Fixture
	problems []string
	pending  int
}

func (q *fixtureQueue) problem(s string) {
	q.mu.Lock()
	q.problems = append(q.problems, s)
	q.mu.Unlock()
}
func requestBody(r *network.Request) string {
	var out strings.Builder
	for _, e := range r.PostDataEntries {
		out.Write(e.Bytes)
	}
	return out.String()
}
func (q *fixtureQueue) intercept(ctx context.Context, e *fetch.EventRequestPaused, targetOrigin, originalOrigin string) {
	q.mu.Lock()
	q.pending++
	q.mu.Unlock()
	defer func() { q.mu.Lock(); q.pending--; q.mu.Unlock() }()
	reqURL, _ := url.Parse(e.Request.URL)
	origin := reqURL.Scheme + "://" + reqURL.Host
	abort := func() {
		_, _ = chromedp.Call(ctx, fetch.FailRequest, fetch.FailRequestParams{RequestID: e.RequestID, ErrorReason: network.ErrorReasonBlockedByClient})
	}
	if e.ResourceType != network.ResourceTypeXHR && e.ResourceType != network.ResourceTypeFetch {
		if origin == targetOrigin && e.Request.Method == "GET" {
			_, _ = chromedp.Call(ctx, fetch.ContinueRequest, fetch.ContinueRequestParams{RequestID: e.RequestID})
			return
		}
		// A favicon is not part of the scenario; other external resources may affect it.
		if !strings.HasSuffix(reqURL.Path, "favicon.ico") {
			q.problem("Blocked resource: " + e.Request.URL)
		}
		abort()
		return
	}
	rawBody := requestBody(e.Request)
	if e.Request.HasPostData && rawBody == "" && e.NetworkID != "" {
		response, err := chromedp.Call(ctx, network.GetRequestPostData, network.GetRequestPostDataParams{RequestID: e.NetworkID})
		if err == nil {
			rawBody = string(response.PostData)
		}
	}
	key, err := casefile.RequestKey(e.Request.Method, e.Request.URL, rawBody)
	if err != nil {
		q.problem(err.Error())
		abort()
		return
	}
	// Only remap API calls belonging to the captured frontend origin. Other API
	// origins retain their full identity even though they are fulfilled locally.
	identity := origin
	if origin == targetOrigin {
		identity = originalOrigin
	}
	key = identity + " " + key
	q.mu.Lock()
	fixtures := q.items[key]
	var f casefile.Fixture
	if len(fixtures) > 0 {
		f = fixtures[0]
		q.items[key] = fixtures[1:]
	}
	q.mu.Unlock()
	if f.ID == "" {
		q.problem("Missing fixture: " + e.Request.Method + " " + e.Request.URL)
		abort()
		return
	}
	if f.DelayMS > 0 {
		select {
		case <-time.After(time.Duration(f.DelayMS) * time.Millisecond):
		case <-ctx.Done():
			return
		}
	}
	_, err = chromedp.Call(ctx, fetch.FulfillRequest, fetch.FulfillRequestParams{RequestID: e.RequestID, ResponseCode: f.Status, ResponseHeaders: []*fetch.HeaderEntry{{Name: "Content-Type", Value: f.ContentType}, {Name: "Access-Control-Allow-Origin", Value: targetOrigin}}, Body: []byte(f.Body)})
	if err != nil {
		q.problem("Could not fulfill fixture: " + err.Error())
	}
}

func Run(parent context.Context, c *casefile.Case, target, variantID string, o Options) Result {
	r := Result{ID: casefile.NewID(), CaseID: c.ID, Target: target, VariantID: variantID, Status: "incomplete", StartedAt: time.Now().UTC(), Steps: []Step{}, Problems: []string{}}
	start := time.Now()
	finish := func() Result { r.DurationMS = time.Since(start).Milliseconds(); return r }
	if err := c.Validate(); err != nil {
		r.Problems = append(r.Problems, err.Error())
		return finish()
	}
	if err := c.Runnable(); err != nil {
		r.Problems = append(r.Problems, err.Error())
		return finish()
	}
	if err := CheckTarget(target); err != nil {
		r.Problems = append(r.Problems, err.Error())
		return finish()
	}
	var variant *casefile.Variant
	if variantID != "" {
		for i := range c.Variants {
			if c.Variants[i].ID == variantID {
				variant = &c.Variants[i]
			}
		}
		if variant == nil {
			r.Problems = append(r.Problems, "variant not found")
			return finish()
		}
	}
	deadline, stopDeadline := context.WithTimeout(parent, 90*time.Second)
	defer stopDeadline()
	ctx, stop, err := launch(deadline, o)
	if err != nil {
		r.Problems = append(r.Problems, err.Error())
		return finish()
	}
	defer stop()
	targetURL, _ := url.Parse(target)
	originalURL, _ := url.Parse(c.Origin)
	targetOrigin := targetURL.Scheme + "://" + targetURL.Host
	originalOrigin := originalURL.Scheme + "://" + originalURL.Host
	q := &fixtureQueue{items: map[string][]casefile.Fixture{}}
	for _, f := range c.Fixtures {
		if variant != nil && f.ID == variant.FixtureID {
			f.Body = variant.Body
			f.Status = variant.Status
			f.DelayMS = variant.DelayMS
		}
		key, _ := casefile.RequestKey(f.Method, f.URL, f.RequestBody)
		u, _ := url.Parse(f.URL)
		key = u.Scheme + "://" + u.Host + " " + key
		q.items[key] = append(q.items[key], f)
	}
	var errorsMu sync.Mutex
	errorCount := 0
	chromedp.Listen(ctx, func(ev any) {
		switch e := ev.(type) {
		case *cdpruntime.EventBindingCalled:
			if e.Name == "__kipurayUnsupported" {
				q.problem(safeMessage(e.Payload))
			}
		case *fetch.EventRequestPaused:
			go q.intercept(ctx, e, targetOrigin, originalOrigin)
		case *cdpruntime.EventExceptionThrown:
			errorsMu.Lock()
			errorCount++
			errorsMu.Unlock()
		}
	})
	err = chromedp.Do(ctx,
		command(cdpruntime.AddBinding, cdpruntime.AddBindingParams{Name: "__kipurayUnsupported"}),
		command(page.AddScriptToEvaluateOnNewDocument, page.AddScriptToEvaluateOnNewDocumentParams{Source: unsupportedAPIs}),
		chromedp.EmulateViewport(int64(c.Width), int64(c.Height)), command(network.Enable, network.EnableParams{}), command(network.SetBypassServiceWorker, network.SetBypassServiceWorkerParams{Bypass: true}), command(network.SetCacheDisabled, network.SetCacheDisabledParams{CacheDisabled: true}), command(fetch.Enable, fetch.EnableParams{Patterns: []*fetch.RequestPattern{{URLPattern: "*", RequestStage: fetch.RequestStageRequest}}}), chromedp.Navigate(target))
	if err != nil {
		r.Problems = append(r.Problems, "Navigation failed: "+err.Error())
		return finish()
	}
	for i, a := range c.Actions {
		if err := perform(ctx, a, targetOrigin); err != nil {
			r.Steps = append(r.Steps, Step{fmt.Sprintf("%d. %s %s", i+1, a.Kind, a.Selector), "incomplete", err.Error()})
			r.Problems = append(r.Problems, "An action could not be completed")
			return finish()
		}
		r.Steps = append(r.Steps, Step{fmt.Sprintf("%d. %s %s", i+1, a.Kind, a.Selector), "passed", ""})
	}
	r.Status = "passed"
	for _, e := range c.Expectations {
		ok := false
		detail := ""
		until := time.Now().Add(3 * time.Second)
		for {
			var evalErr error
			if e.Kind == "no-errors" {
				errorsMu.Lock()
				ok = errorCount == 0
				errorsMu.Unlock()
			} else {
				ok, evalErr = evaluateExpectation(ctx, e)
			}
			if evalErr != nil {
				r.Status = "incomplete"
				detail = evalErr.Error()
				break
			}
			if ok || time.Now().After(until) {
				break
			}
			select {
			case <-time.After(50 * time.Millisecond):
			case <-ctx.Done():
				detail = ctx.Err().Error()
				r.Status = "incomplete"
			}
			if detail != "" {
				break
			}
		}
		status := "passed"
		if !ok {
			status = "failed"
			if r.Status != "incomplete" {
				r.Status = "failed"
			}
			if detail == "" {
				detail = "Expected condition was not satisfied"
			}
		}
		r.Steps = append(r.Steps, Step{e.Kind + " " + e.Selector + " " + e.Value, status, detail})
	}
	// Drain intercepted requests so a missing fixture cannot produce a false pass.
	until := time.Now().Add(time.Second)
	for time.Now().Before(until) {
		time.Sleep(50 * time.Millisecond)
		q.mu.Lock()
		pending := q.pending
		q.mu.Unlock()
		if pending == 0 {
			break
		}
	}
	q.mu.Lock()
	r.Problems = append(r.Problems, q.problems...)
	if q.pending > 0 {
		r.Problems = append(r.Problems, "Requests still pending")
	}
	for _, remaining := range q.items {
		if len(remaining) > 0 {
			r.Problems = append(r.Problems, "Recorded API requests were not consumed")
			break
		}
	}
	q.mu.Unlock()
	if len(r.Problems) > 0 {
		r.Status = "incomplete"
	}
	shot, shotErr := chromedp.Run(ctx, chromedp.CaptureScreenshot())
	if shotErr == nil {
		r.Screenshot = base64.StdEncoding.EncodeToString(shot)
	}
	return finish()
}

func quote(s string) string { b, _ := json.Marshal(s); return string(b) }
func perform(ctx context.Context, a casefile.Action, targetOrigin string) error {
	if a.Kind == "navigate" {
		u, err := url.Parse(a.Value)
		if err != nil {
			return err
		}
		return chromedp.Do(ctx, chromedp.Navigate(targetOrigin+u.RequestURI()))
	}
	until := time.Now().Add(5 * time.Second)
	for {
		count, err := chromedp.Run(ctx, chromedp.Evaluate[int](`document.querySelectorAll(`+quote(a.Selector)+`).length`))
		if err != nil {
			return err
		}
		if count > 1 {
			return fmt.Errorf("selector matched %d elements", count)
		}
		if count == 1 {
			break
		}
		if time.Now().After(until) {
			return fmt.Errorf("element not found: %s", a.Selector)
		}
		select {
		case <-time.After(50 * time.Millisecond):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	if a.Kind == "click" {
		return chromedp.Do(ctx, chromedp.Click(chromedp.CSS(a.Selector)))
	}
	script := `(()=>{const el=document.querySelector(` + quote(a.Selector) + `);const proto=el instanceof HTMLTextAreaElement?HTMLTextAreaElement.prototype:el instanceof HTMLSelectElement?HTMLSelectElement.prototype:HTMLInputElement.prototype;const setter=Object.getOwnPropertyDescriptor(proto,'value').set;setter.call(el,` + quote(a.Value) + `);el.dispatchEvent(new Event('input',{bubbles:true}));el.dispatchEvent(new Event('change',{bubbles:true}));})()`
	return chromedp.Do(ctx, chromedp.Evaluate[chromedp.Void](script))
}
func evaluateExpectation(ctx context.Context, e casefile.Expectation) (bool, error) {
	script := ""
	switch e.Kind {
	case "url":
		script = `location.pathname+location.search===` + quote(e.Value)
	case "visible":
		script = `(()=>{const e=document.querySelector(` + quote(e.Selector) + `);return !!e&&e.getClientRects().length>0&&getComputedStyle(e).visibility!=='hidden'})()`
	case "text":
		script = `(()=>{const e=document.querySelector(` + quote(e.Selector) + `);return !!e&&e.textContent.includes(` + quote(e.Value) + `)})()`
	}
	return chromedp.Run(ctx, chromedp.Evaluate[bool](script))
}

// Unsupported network paths fail before a worker or streaming connection can
// escape the fixture interceptor. They must not silently produce green checks.
const unsupportedAPIs = `(()=>{
const report=name=>{window.__kipurayUnsupported("Unsupported browser API: "+name);return new Error(name+" is outside this scenario's supported scope")};
for(const name of ['WebSocket','EventSource','Worker','SharedWorker','WebTransport']){if(name in window){window[name]=class{constructor(){throw report(name)}}}}
navigator.sendBeacon=()=>{report('sendBeacon');return false};
if(navigator.serviceWorker){navigator.serviceWorker.register=()=>Promise.reject(report('serviceWorker'))}
})()`
