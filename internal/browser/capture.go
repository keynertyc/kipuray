package browser

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/fetch"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/page"
	cdpruntime "github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
	"github.com/keynertyc/kipuray/internal/casefile"
)

type Capture struct {
	mu          sync.Mutex
	c           *casefile.Case
	cancel      context.CancelFunc
	ctx         context.Context
	bytes       int
	bodies      bool
	stopped     bool
	workers     sync.WaitGroup
	requestKeys map[network.RequestID]string
	activeKeys  map[string]int
}
type CaptureOptions struct {
	Options
	CaptureJSON bool
}

func StartCapture(parent context.Context, target string, o CaptureOptions) (*Capture, error) {
	if err := CheckTarget(target); err != nil {
		return nil, err
	}
	deadline, stopDeadline := context.WithTimeout(parent, 10*time.Minute)
	ctx, stop, err := launch(deadline, o.Options)
	if err != nil {
		stopDeadline()
		return nil, err
	}
	sanitizedOrigin, _ := safeURL(target)
	s := &Capture{c: casefile.New(sanitizedOrigin), ctx: ctx, bodies: o.CaptureJSON, cancel: func() { stop(); stopDeadline() }}
	s.requestKeys = map[network.RequestID]string{}
	s.activeKeys = map[string]int{}
	s.c.RecordingState = "recording"
	s.c.Warnings = append(s.c.Warnings, "Use synthetic test data. Review all payloads, page text, and diagnostics before sharing.", "External visual resources are blocked during playback; unavailable assets may appear blank.")
	chromedp.Listen(ctx, func(ev any) {
		switch e := ev.(type) {
		case *network.EventRequestWillBeSent:
			if e.Type == network.ResourceTypeXHR || e.Type == network.ResourceTypeFetch {
				key, _ := casefile.RequestKey(e.Request.Method, e.Request.URL, requestBody(e.Request))
				key = e.Request.URL + key
				s.mu.Lock()
				if old := s.requestKeys[e.RequestID]; old != "" {
					s.activeKeys[old]--
				}
				if s.activeKeys[key] > 0 && len(s.c.ExecutionProblems) < 20 {
					s.c.ExecutionProblems = append(s.c.ExecutionProblems, "Concurrent identical API requests are not supported by this scenario format")
				}
				s.activeKeys[key]++
				s.requestKeys[e.RequestID] = key
				s.mu.Unlock()
			}
		case *network.EventLoadingFinished:
			s.finishRequest(e.RequestID)
		case *network.EventLoadingFailed:
			s.finishRequest(e.RequestID)
		case *network.EventWebSocketCreated:
			s.mu.Lock()
			s.c.ExecutionProblems = append(s.c.ExecutionProblems, "WebSocket traffic is outside the supported scope")
			s.mu.Unlock()

		case *cdpruntime.EventBindingCalled:
			if e.Name == "__kipurayEmit" {
				s.receive(e.Payload)
			}
		case *fetch.EventRequestPaused:
			s.mu.Lock()
			if !s.stopped {
				s.workers.Add(1)
				go func() { defer s.workers.Done(); s.response(e) }()
			}
			s.mu.Unlock()
		case *cdpruntime.EventConsoleAPICalled:
			parts := []string{}
			for _, arg := range e.Args {
				var value any
				if json.Unmarshal(arg.Value, &value) == nil {
					switch x := value.(type) {
					case string:
						parts = append(parts, x)
					case float64, bool:
						parts = append(parts, fmt.Sprint(x))
					}
				}
			}
			if len(parts) > 0 {
				s.diagnostic(string(e.Type), strings.Join(parts, " "))
			}
		case *cdpruntime.EventExceptionThrown:
			message := e.ExceptionDetails.Text
			if e.ExceptionDetails.Exception != nil && e.ExceptionDetails.Exception.Description != "" {
				message = e.ExceptionDetails.Exception.Description
			}
			s.diagnostic("error", message)
		}
	})
	err = chromedp.Do(ctx,
		command(cdpruntime.Enable, cdp.Empty{}),
		command(cdpruntime.AddBinding, cdpruntime.AddBindingParams{Name: "__kipurayEmit"}),
		command(page.AddScriptToEvaluateOnNewDocument, page.AddScriptToEvaluateOnNewDocumentParams{Source: recorder}),
		command(network.Enable, network.EnableParams{}),
		command(fetch.Enable, fetch.EnableParams{Patterns: []*fetch.RequestPattern{{ResourceType: network.ResourceTypeXHR, RequestStage: fetch.RequestStageResponse}, {ResourceType: network.ResourceTypeFetch, RequestStage: fetch.RequestStageResponse}}}),
		chromedp.Navigate(target))
	if err != nil {
		s.cancel()
		return nil, err
	}
	return s, nil
}
func (s *Capture) receive(payload string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stopped {
		return
	}
	s.bytes += len(payload)
	if s.bytes > casefile.MaxBytes/2 {
		s.c.Truncated = true
		if len(s.c.Warnings) < 10 {
			s.c.Warnings = append(s.c.Warnings, "Recording size limit reached. Stop and create a shorter case.")
		}
		return
	}
	var msg struct {
		Kind    string          `json:"kind"`
		Event   json.RawMessage `json:"event"`
		Action  casefile.Action `json:"action"`
		Message string          `json:"message"`
	}
	if json.Unmarshal([]byte(payload), &msg) != nil {
		return
	}
	switch msg.Kind {
	case "event":
		s.c.Events = append(s.c.Events, msg.Event)
	case "action":
		s.c.Actions = append(s.c.Actions, msg.Action)
	case "error":
		s.c.Diagnostics = append(s.c.Diagnostics, casefile.Diagnostic{Kind: "error", Message: safeMessage(msg.Message), At: time.Now().UnixMilli()})
	}
}
func (s *Capture) response(e *fetch.EventRequestPaused) {
	defer func() {
		_, _ = chromedp.Call(s.ctx, fetch.ContinueRequest, fetch.ContinueRequestParams{RequestID: e.RequestID})
	}()
	if e.ResponseStatusCode < 100 {
		return
	}
	ct := ""
	for _, h := range e.ResponseHeaders {
		if strings.EqualFold(h.Name, "content-type") {
			ct = h.Value
		}
	}
	rawURL, urlRedacted := safeURL(e.Request.URL)
	f := casefile.Fixture{ID: casefile.NewID(), Method: e.Request.Method, URL: rawURL, Status: e.ResponseStatusCode, ContentType: ct, Redacted: !s.bodies || urlRedacted}
	if e.Request.HasPostData {
		f.Redacted = true
		if s.bodies && e.NetworkID != "" {
			response, err := chromedp.Call(s.ctx, network.GetRequestPostData, network.GetRequestPostDataParams{RequestID: e.NetworkID})
			if err == nil {
				var redacted bool
				f.RequestBody, redacted = ScrubJSON(string(response.PostData))
				f.Redacted = redacted || urlRedacted
			}
		}
	}
	if s.bodies && strings.Contains(ct, "json") {
		response, err := chromedp.Call(s.ctx, fetch.GetResponseBody, fetch.GetResponseBodyParams{RequestID: e.RequestID})
		var b []byte
		if err == nil {
			b = response.Body
		}
		if err == nil && len(b) <= 2<<20 {
			body, redacted := ScrubJSON(string(b))
			f.Body = body
			f.Redacted = f.Redacted || redacted
		} else {
			f.Redacted = true
		}
	}
	if f.Body == "" {
		f.Redacted = true
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.bytes+len(f.Body)+len(f.RequestBody) > casefile.MaxBytes/2 || len(s.c.Fixtures) >= 2000 {
		s.c.Truncated = true
		return
	}
	s.bytes += len(f.Body) + len(f.RequestBody)
	s.c.Fixtures = append(s.c.Fixtures, f)
}
func (s *Capture) Snapshot() *casefile.Case {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, _ := json.Marshal(s.c)
	var c casefile.Case
	_ = json.Unmarshal(b, &c)
	return &c
}
func (s *Capture) Stop() *casefile.Case {
	s.mu.Lock()
	s.stopped = true
	s.c.RecordingState = "complete"
	s.mu.Unlock()
	s.cancel()
	s.workers.Wait()
	return s.Snapshot()
}
func (s *Capture) Status() string {
	if s.ctx.Err() != nil {
		return fmt.Sprint("stopped: ", s.ctx.Err())
	}
	return "recording"
}

func (s *Capture) diagnostic(kind, message string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.stopped && len(s.c.Diagnostics) < 1000 {
		s.c.Diagnostics = append(s.c.Diagnostics, casefile.Diagnostic{Kind: kind, Message: safeMessage(message), At: time.Now().UnixMilli()})
	}
}

func (s *Capture) finishRequest(id network.RequestID) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if key := s.requestKeys[id]; key != "" {
		s.activeKeys[key]--
		delete(s.requestKeys, id)
	}
}
