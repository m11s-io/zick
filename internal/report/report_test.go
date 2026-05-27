package report

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/m11s-io/zick/internal/fresh"
)

func TestWriteJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "zick-report.json")
	rep := New(".")
	rep.Fresh.Status = "passed"
	rep.Fresh.Results = FreshResults([]fresh.Result{{
		Package:   "lodash",
		Version:   "4.17.21",
		Published: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		Age:       24 * time.Hour,
		Risk:      fresh.RiskHigh,
	}})
	Finalize(&rep, time.Now().Add(-time.Second))

	if err := WriteJSON(path, rep); err != nil {
		t.Fatalf("WriteJSON: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	got := string(data)
	if !strings.Contains(got, `"schema_version": "1"`) || !strings.Contains(got, `"risk": "HIGH"`) {
		t.Fatalf("json report = %s, want schema and fresh result", got)
	}
}

func TestWriteHTML(t *testing.T) {
	path := filepath.Join(t.TempDir(), "zick-report.html")
	rep := New(".")
	rep.Status = "failed"
	rep.Fresh.Status = "failed"
	rep.Fresh.Results = []FreshResult{{
		Risk:      "WARN",
		Package:   "left-pad",
		Version:   "1.3.0",
		Published: "2025-01-01T00:00:00Z",
		Age:       "1d",
	}}
	Finalize(&rep, time.Now().Add(-time.Second))

	if err := WriteHTML(path, rep); err != nil {
		t.Fatalf("WriteHTML: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	got := string(data)
	if !strings.Contains(got, "<title>zick report</title>") || !strings.Contains(got, "left-pad") {
		t.Fatalf("html report = %s, want title and package", got)
	}
}
