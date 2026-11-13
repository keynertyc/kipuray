package browser

import (
	"encoding/json"
	"net/url"
	"regexp"
	"strings"
)

var sensitive = regexp.MustCompile(`(?i)(password|passwd|secret|token|authorization|cookie|api.?key|session.?id)`)

// ScrubJSON excludes recognized credential fields. This is a conservative aid,
// not an automatic guarantee of anonymization; test payloads still need review.
func ScrubJSON(body string) (string, bool) {
	var v any
	if json.Unmarshal([]byte(body), &v) != nil {
		return "", true
	}
	changed := false
	var walk func(any)
	walk = func(n any) {
		switch x := n.(type) {
		case map[string]any:
			for k, v := range x {
				if sensitive.MatchString(k) {
					x[k] = "[REDACTED]"
					changed = true
				} else {
					walk(v)
				}
			}
		case []any:
			for _, v := range x {
				walk(v)
			}
		}
	}
	walk(v)
	b, _ := json.Marshal(v)
	return string(b), changed
}
func safeURL(raw string) (string, bool) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", true
	}
	q := u.Query()
	changed := false
	for k := range q {
		if sensitive.MatchString(k) {
			q.Set(k, "[REDACTED]")
			changed = true
		}
	}
	u.User = nil
	u.RawQuery = q.Encode()
	u.Fragment = ""
	return u.String(), changed
}
func safeMessage(s string) string {
	if sensitive.MatchString(s) {
		return "Diagnostic omitted: may contain credentials"
	}
	if len(s) > 2000 {
		return s[:2000]
	}
	return strings.ToValidUTF8(s, "")
}
