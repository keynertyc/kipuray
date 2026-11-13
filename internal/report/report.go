// Package report renders machine-readable and self-contained CI evidence.
package report

import (
	"encoding/json"
	"encoding/xml"
	"io"

	"github.com/keynertyc/kipuray/internal/browser"
)

func Write(w io.Writer, format string, value any) error {
	switch format {
	case "junit":
		return junit(w, value)
	case "html":
		return writeHTML(w, value)
	default:
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(value)
	}
}

type testSuite struct {
	XMLName  xml.Name   `xml:"testsuite"`
	Name     string     `xml:"name,attr"`
	Tests    int        `xml:"tests,attr"`
	Failures int        `xml:"failures,attr"`
	Errors   int        `xml:"errors,attr"`
	Cases    []testCase `xml:"testcase"`
}
type testCase struct {
	Name    string  `xml:"name,attr"`
	Failure *string `xml:"failure,omitempty"`
	Error   *string `xml:"error,omitempty"`
}

func junit(w io.Writer, v any) error {
	s := testSuite{Name: "Kipuray"}
	var status string
	switch x := v.(type) {
	case browser.Result:
		status = x.Status
	case browser.Comparison:
		status = x.Status
	}
	tc := testCase{Name: "Recorded scenario"}
	if status == "incomplete" {
		message := "Execution incomplete: inspect JSON evidence"
		tc.Error = &message
		s.Errors = 1
	} else if status != "passed" && status != "verified" {
		message := status
		tc.Failure = &message
		s.Failures = 1
	}
	s.Tests = 1
	s.Cases = append(s.Cases, tc)
	return xml.NewEncoder(w).Encode(s)
}
