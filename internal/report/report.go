package report

import (
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"time"

	"github.com/m11s-io/zick/internal/fresh"
)

type Report struct {
	SchemaVersion string       `json:"schema_version"`
	GeneratedAt   string       `json:"generated_at"`
	Target        string       `json:"target"`
	Status        string       `json:"status"`
	Duration      string       `json:"duration"`
	Summary       Summary      `json:"summary"`
	Fresh         FreshSection `json:"fresh"`
	Secrets       ToolSection  `json:"secrets"`
	Scan          ToolSection  `json:"scan"`
}

type Summary struct {
	FreshHigh   int `json:"fresh_high"`
	FreshWarn   int `json:"fresh_warn"`
	FreshOK     int `json:"fresh_ok"`
	CheckPassed int `json:"check_passed"`
	CheckFailed int `json:"check_failed"`
	Skipped     int `json:"skipped"`
}

type FreshSection struct {
	Status       string        `json:"status"`
	Duration     string        `json:"duration"`
	AgeGateDays  int           `json:"age_gate_days"`
	IncludeDev   bool          `json:"include_dev"`
	FailOn       string        `json:"fail_on"`
	Error        string        `json:"error,omitempty"`
	Results      []FreshResult `json:"results"`
	NoManifest   bool          `json:"no_manifest"`
	ViolationCnt int           `json:"violation_count"`
}

type FreshResult struct {
	Risk      string `json:"risk"`
	Package   string `json:"package"`
	Version   string `json:"version"`
	Published string `json:"published"`
	Age       string `json:"age"`
}

type ToolSection struct {
	Status   string   `json:"status"`
	Duration string   `json:"duration"`
	Tools    []string `json:"tools,omitempty"`
	Tool     string   `json:"tool,omitempty"`
	Output   string   `json:"output,omitempty"`
	Error    string   `json:"error,omitempty"`
}

func New(target string) Report {
	return Report{
		SchemaVersion: "1",
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
		Target:        target,
		Status:        "passed",
		Fresh:         FreshSection{Status: "skipped"},
		Secrets:       ToolSection{Status: "skipped"},
		Scan:          ToolSection{Status: "skipped"},
	}
}

func FreshResults(results []fresh.Result) []FreshResult {
	rows := make([]FreshResult, 0, len(results))
	for _, r := range results {
		rows = append(rows, FreshResult{
			Risk:      riskLabel(r.Risk),
			Package:   r.Package,
			Version:   r.Version,
			Published: r.Published.Format(time.RFC3339),
			Age:       humanAge(r.Age),
		})
	}
	return rows
}

func Finalize(r *Report, started time.Time) {
	r.Duration = time.Since(started).Round(time.Millisecond).String()

	var failed bool
	for _, row := range r.Fresh.Results {
		switch row.Risk {
		case "HIGH":
			r.Summary.FreshHigh++
		case "WARN":
			r.Summary.FreshWarn++
		default:
			r.Summary.FreshOK++
		}
	}

	for _, status := range []string{r.Fresh.Status, r.Secrets.Status, r.Scan.Status} {
		switch status {
		case "failed":
			failed = true
			r.Summary.CheckFailed++
		case "passed":
			r.Summary.CheckPassed++
		case "skipped":
			r.Summary.Skipped++
		}
	}
	if failed {
		r.Status = "failed"
	} else {
		r.Status = "passed"
	}
}

func WriteJSON(path string, r Report) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil && filepath.Dir(path) != "." {
		return fmt.Errorf("create report directory: %w", err)
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create %s: %w", path, err)
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(r); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

func WriteHTML(path string, r Report) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil && filepath.Dir(path) != "." {
		return fmt.Errorf("create report directory: %w", err)
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create %s: %w", path, err)
	}
	defer f.Close()

	tpl, err := template.New("report").Funcs(template.FuncMap{
		"lower": lower,
	}).Parse(htmlTemplate)
	if err != nil {
		return fmt.Errorf("parse report template: %w", err)
	}
	if err := tpl.Execute(f, r); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

func riskLabel(r fresh.Risk) string {
	switch r {
	case fresh.RiskHigh:
		return "HIGH"
	case fresh.RiskWarn:
		return "WARN"
	default:
		return "OK"
	}
}

func humanAge(d time.Duration) string {
	days := int(d.Hours() / 24)
	if days > 0 {
		return fmt.Sprintf("%dd", days)
	}
	hours := int(d.Hours())
	if hours > 0 {
		return fmt.Sprintf("%dh", hours)
	}
	return d.Round(time.Minute).String()
}

func lower(s string) string {
	switch s {
	case "HIGH":
		return "high"
	case "WARN":
		return "warn"
	case "OK":
		return "ok"
	default:
		return s
	}
}

const htmlTemplate = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>zick report</title>
<style>
:root{color-scheme:light;--bg:#f7f8fa;--panel:#fff;--text:#17202a;--muted:#667085;--line:#d9dee7;--ok:#16794c;--warn:#a05a00;--high:#b42318;--skip:#596579}
*{box-sizing:border-box}body{margin:0;background:var(--bg);color:var(--text);font:14px/1.5 system-ui,-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif}
main{max-width:1120px;margin:0 auto;padding:32px 20px 48px}.top{display:flex;justify-content:space-between;gap:20px;align-items:flex-start;margin-bottom:24px}
h1{font-size:28px;margin:0 0 4px}h2{font-size:18px;margin:0 0 12px}.meta{color:var(--muted);font-size:13px}.badge{display:inline-flex;align-items:center;border:1px solid var(--line);border-radius:999px;padding:4px 10px;font-weight:650;background:var(--panel)}
.passed{color:var(--ok)}.failed,.high{color:var(--high)}.warn{color:var(--warn)}.skipped{color:var(--skip)}.ok{color:var(--ok)}
.grid{display:grid;grid-template-columns:repeat(4,minmax(0,1fr));gap:12px;margin-bottom:18px}.metric,.section{background:var(--panel);border:1px solid var(--line);border-radius:8px}
.metric{padding:14px}.metric strong{display:block;font-size:24px}.metric span{color:var(--muted);font-size:12px;text-transform:uppercase;letter-spacing:.04em}
.section{padding:18px;margin-top:14px;overflow:hidden}.section-head{display:flex;justify-content:space-between;gap:12px;align-items:center;margin-bottom:10px}
table{width:100%;border-collapse:collapse;font-size:13px}th,td{text-align:left;border-top:1px solid var(--line);padding:9px 8px;vertical-align:top}th{color:var(--muted);font-size:12px;text-transform:uppercase;letter-spacing:.04em}
pre{white-space:pre-wrap;overflow:auto;background:#111827;color:#e5e7eb;border-radius:6px;padding:12px;margin:10px 0 0;font-size:12px;max-height:360px}
.empty{color:var(--muted);padding:8px 0}@media (max-width:760px){.top{display:block}.grid{grid-template-columns:repeat(2,minmax(0,1fr))}main{padding:22px 14px 36px}}
</style>
</head>
<body>
<main>
<div class="top">
  <div><h1>zick report</h1><div class="meta">{{.Target}} &middot; {{.GeneratedAt}} &middot; {{.Duration}}</div></div>
  <div class="badge {{.Status}}">{{.Status}}</div>
</div>
<div class="grid">
  <div class="metric"><strong class="high">{{.Summary.FreshHigh}}</strong><span>fresh high</span></div>
  <div class="metric"><strong class="warn">{{.Summary.FreshWarn}}</strong><span>fresh warn</span></div>
  <div class="metric"><strong class="ok">{{.Summary.CheckPassed}}</strong><span>passed checks</span></div>
  <div class="metric"><strong class="failed">{{.Summary.CheckFailed}}</strong><span>failed checks</span></div>
</div>
<section class="section">
  <div class="section-head"><h2>Freshness</h2><span class="badge {{.Fresh.Status}}">{{.Fresh.Status}}</span></div>
  <div class="meta">age gate {{.Fresh.AgeGateDays}}d &middot; fail on {{.Fresh.FailOn}} &middot; {{.Fresh.Duration}}</div>
  {{if .Fresh.Error}}<pre>{{.Fresh.Error}}</pre>{{end}}
  {{if .Fresh.NoManifest}}<div class="empty">No supported manifest found.</div>{{end}}
  {{if .Fresh.Results}}
  <table><thead><tr><th>Risk</th><th>Package</th><th>Version</th><th>Published</th><th>Age</th></tr></thead><tbody>
  {{range .Fresh.Results}}<tr><td class="{{lower .Risk}}">{{.Risk}}</td><td>{{.Package}}</td><td>{{.Version}}</td><td>{{.Published}}</td><td>{{.Age}}</td></tr>{{end}}
  </tbody></table>
  {{end}}
</section>
<section class="section">
  <div class="section-head"><h2>Secrets</h2><span class="badge {{.Secrets.Status}}">{{.Secrets.Status}}</span></div>
  <div class="meta">{{.Secrets.Tool}} &middot; {{.Secrets.Duration}}</div>
  {{if .Secrets.Error}}<pre>{{.Secrets.Error}}</pre>{{end}}{{if .Secrets.Output}}<pre>{{.Secrets.Output}}</pre>{{end}}
</section>
<section class="section">
  <div class="section-head"><h2>Vulnerabilities</h2><span class="badge {{.Scan.Status}}">{{.Scan.Status}}</span></div>
  <div class="meta">{{range $i,$t := .Scan.Tools}}{{if $i}}, {{end}}{{$t}}{{end}} &middot; {{.Scan.Duration}}</div>
  {{if .Scan.Error}}<pre>{{.Scan.Error}}</pre>{{end}}{{if .Scan.Output}}<pre>{{.Scan.Output}}</pre>{{end}}
</section>
</main>
</body>
</html>
`
