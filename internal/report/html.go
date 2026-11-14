package report

import (
	_ "embed"
	"fmt"
	"html/template"
	"io"

	"github.com/keynertyc/kipuray/internal/browser"
)

type htmlEvidence struct {
	Status string
	Runs   []browser.Result
}

func writeHTML(w io.Writer, value any) error {
	data := htmlEvidence{}
	switch v := value.(type) {
	case browser.Result:
		data.Status = v.Status
		data.Runs = []browser.Result{v}
	case browser.Comparison:
		data.Status = v.Status
		data.Runs = []browser.Result{v.Before, v.After}
	default:
		return fmt.Errorf("unsupported report type %T", value)
	}
	return htmlReport.Execute(w, data)
}

func parseHTMLReport(source string) (*template.Template, error) {
	return template.New("report").Funcs(template.FuncMap{"screenshot": func(b64 string) template.URL { return template.URL("data:image/png;base64," + b64) }}).Parse(source)
}

var htmlReport = template.Must(parseHTMLReport(reportTemplate))

//go:embed report.html
var reportTemplate string
