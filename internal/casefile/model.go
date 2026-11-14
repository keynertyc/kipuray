// Package casefile defines the portable, versioned contract shared by capture,
// playback, and execution. It contains data only; cases never embed executable code.
package casefile

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"
)

const Version = 1
const MaxBytes = 64 << 20

type Case struct {
	ExecutionProblems []string          `json:"executionProblems,omitempty"`
	RecordingState    string            `json:"recordingState,omitempty"`
	Truncated         bool              `json:"truncated,omitempty"`
	Version           int               `json:"version"`
	ID                string            `json:"id"`
	Title             string            `json:"title"`
	CreatedAt         time.Time         `json:"createdAt"`
	Origin            string            `json:"origin"`
	Width             int               `json:"width"`
	Height            int               `json:"height"`
	Events            []json.RawMessage `json:"events"`
	Actions           []Action          `json:"actions"`
	Fixtures          []Fixture         `json:"fixtures"`
	Expectations      []Expectation     `json:"expectations"`
	Diagnostics       []Diagnostic      `json:"diagnostics"`
	Warnings          []string          `json:"warnings"`
	Variants          []Variant         `json:"variants"`
}
type Action struct {
	Kind     string `json:"kind"`
	Selector string `json:"selector,omitempty"`
	Value    string `json:"value,omitempty"`
	At       int64  `json:"at"`
	Redacted bool   `json:"redacted,omitempty"`
}
type Fixture struct {
	ID          string `json:"id"`
	Method      string `json:"method"`
	URL         string `json:"url"`
	RequestBody string `json:"requestBody,omitempty"`
	Status      int64  `json:"status"`
	Body        string `json:"body"`
	ContentType string `json:"contentType"`
	DelayMS     int    `json:"delayMs"`
	Redacted    bool   `json:"redacted,omitempty"`
}
type Expectation struct {
	Kind     string `json:"kind"`
	Selector string `json:"selector,omitempty"`
	Value    string `json:"value,omitempty"`
}
type Diagnostic struct {
	Kind    string `json:"kind"`
	Message string `json:"message"`
	At      int64  `json:"at"`
}
type Variant struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	FixtureID string `json:"fixtureId"`
	Body      string `json:"body"`
	Status    int64  `json:"status"`
	DelayMS   int    `json:"delayMs"`
}

func New(origin string) *Case {
	return &Case{Version: Version, ID: NewID(), Title: "Untitled case", CreatedAt: time.Now().UTC(), Origin: origin, Width: 1280, Height: 800, Events: []json.RawMessage{}, Actions: []Action{}, Fixtures: []Fixture{}, Expectations: []Expectation{}, Diagnostics: []Diagnostic{}, Warnings: []string{}, Variants: []Variant{}}
}
func NewID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b[:])
}
func ValidID(id string) bool { b, err := hex.DecodeString(id); return err == nil && len(b) == 16 }

func (c *Case) Validate() error {
	if c.Version != Version {
		return fmt.Errorf("unsupported case version %d", c.Version)
	}
	if !ValidID(c.ID) {
		return errors.New("invalid case ID")
	}
	if len(c.Title) == 0 || len(c.Title) > 200 {
		return errors.New("title must be 1–200 characters")
	}
	u, err := url.Parse(c.Origin)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil {
		return errors.New("case origin must be an HTTP URL without credentials")
	}
	if c.Width < 320 || c.Width > 3840 || c.Height < 200 || c.Height > 2160 {
		return errors.New("unsupported viewport")
	}
	if len(c.Actions) > 10000 || len(c.Events) > 200000 || len(c.Fixtures) > 2000 || len(c.Variants) > 2000 || len(c.Expectations) > 200 {
		return errors.New("case exceeds event limits")
	}
	for _, a := range c.Actions {
		if a.Kind != "click" && a.Kind != "input" && a.Kind != "select" && a.Kind != "navigate" {
			return fmt.Errorf("unsupported action %q", a.Kind)
		}
		if a.Kind != "navigate" && a.Selector == "" {
			return errors.New("action needs a selector")
		}
		if len(a.Selector) > 2000 || len(a.Value) > 100000 {
			return errors.New("action too large")
		}
	}
	ids := map[string]bool{}
	for _, f := range c.Fixtures {
		if !ValidID(f.ID) || ids[f.ID] {
			return errors.New("invalid or duplicate fixture ID")
		}
		ids[f.ID] = true
		if f.Status < 100 || f.Status > 599 || f.DelayMS < 0 || f.DelayMS > 30000 {
			return errors.New("invalid fixture status or delay")
		}
		if _, err := RequestKey(f.Method, f.URL, f.RequestBody); err != nil {
			return err
		}
	}
	for _, e := range c.Expectations {
		if (e.Kind == "text" || e.Kind == "url") && e.Value == "" {
			return errors.New("expectation needs a nonempty value")
		}
		switch e.Kind {
		case "visible", "text":
			if e.Selector == "" {
				return errors.New("expectation needs a selector")
			}
		case "url", "no-errors":
		default:
			return fmt.Errorf("unsupported expectation %q", e.Kind)
		}
	}
	for _, v := range c.Variants {
		if !ValidID(v.ID) || !ids[v.FixtureID] || v.Name == "" || v.Status < 100 || v.Status > 599 || v.DelayMS < 0 || v.DelayMS > 30000 {
			return errors.New("invalid variant")
		}
	}
	return nil
}
func (c *Case) Runnable() error {
	if len(c.ExecutionProblems) > 0 {
		return errors.New(strings.Join(c.ExecutionProblems, "; "))
	}
	if c.Truncated {
		return errors.New("recording exceeded its size limit; capture a shorter scenario")
	}
	if c.RecordingState == "recording" {
		return errors.New("recording was interrupted; capture a new complete scenario")
	}
	if len(c.Expectations) == 0 {
		return errors.New("add an expected result before running")
	}
	functional := false
	for _, e := range c.Expectations {
		if e.Kind != "no-errors" {
			functional = true
		}
	}
	if !functional {
		return errors.New("add at least one functional expectation")
	}
	for _, a := range c.Actions {
		if a.Redacted {
			return errors.New("replace masked action values with synthetic test data")
		}
	}
	for _, f := range c.Fixtures {
		if f.Redacted {
			return errors.New("review redacted fixtures before execution")
		}
	}
	return nil
}

// RequestKey deliberately ignores the origin to allow an explicitly chosen local
// target. Occurrences are consumed in order by the runner.
func RequestKey(method, rawURL, body string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return "", errors.New("invalid fixture URL")
	}
	u.RawQuery = u.Query().Encode()
	var value any
	decoder := json.NewDecoder(strings.NewReader(body))
	decoder.UseNumber()
	if decoder.Decode(&value) == nil && decoder.Decode(new(any)) == io.EOF {
		normalized, _ := json.Marshal(value)
		body = string(normalized)
	}
	return strings.ToUpper(method) + " " + u.EscapedPath() + "?" + u.RawQuery + " " + body, nil
}
