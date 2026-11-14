package report

import (
	"bytes"
	"github.com/keynertyc/kipuray/internal/browser"
	"strings"
	"testing"
)

func TestReports(t *testing.T) {
	for _, tc := range []struct{ status, element string }{{"passed", ""}, {"failed", "<failure>"}, {"incomplete", "<error>"}} {
		var b bytes.Buffer
		if err := Write(&b, "junit", browser.Result{Status: tc.status}); err != nil {
			t.Fatal(err)
		}
		if tc.element != "" && !strings.Contains(b.String(), tc.element) {
			t.Fatal(b.String())
		}
	}
	var b bytes.Buffer
	r := browser.Result{Status: "failed", Target: `<script>alert(1)</script>`, Steps: []browser.Step{{Label: "Expected checkout", Status: "failed"}}}
	if err := Write(&b, "html", r); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(b.String(), r.Target) {
		t.Fatal("unescaped user text")
	}
	if !strings.Contains(b.String(), "Expected checkout") {
		t.Fatal("missing evidence")
	}
}

func TestHTMLReportSupportsWindowsLineEndings(t *testing.T) {
	source := strings.ReplaceAll(reportTemplate, "\r\n", "\n")
	report, err := parseHTMLReport(strings.ReplaceAll(source, "\n", "\r\n"))
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	err = report.Execute(&output, htmlEvidence{Status: "passed", Runs: []browser.Result{{Status: "passed", Target: "http://localhost:3000"}}})
	if err != nil {
		t.Fatalf("CRLF template cannot render: %v", err)
	}
	if !strings.Contains(output.String(), "http://localhost:3000") {
		t.Fatal("missing run evidence")
	}
}
