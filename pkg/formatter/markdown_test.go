package formatter

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/sakurahilljp/git-dirstat/pkg/model"
)

func TestMarkdownFormatter(t *testing.T) {
	report := &model.Report{
		Target: ".",
		Depth:  1,
		Entries: []model.Entry{
			{Path: "src/user_service", Files: 3, Added: 10, Deleted: 5, Net: 5, Percent: 50.0},
			{Path: "docs", Files: 1, Added: 0, Deleted: 10, Net: -10, Percent: 33.3},
		},
		Summary: model.Summary{
			TotalFiles:   4,
			TotalAdded:   10,
			TotalDeleted: 15,
			Net:          -5,
		},
	}

	tests := []struct {
		name        string
		showPercent bool
		showGraph   bool
		checkFunc   func(t *testing.T, output string)
	}{
		{
			name:        "default options",
			showPercent: false,
			showGraph:   false,
			checkFunc: func(t *testing.T, output string) {
				if !strings.Contains(output, "**Target:** `.` (Depth: 1)") {
					t.Errorf("missing target line: %s", output)
				}
				if !strings.Contains(output, "| Directory | Files | Added | Deleted | Net |") {
					t.Errorf("missing header: %s", output)
				}
				if !strings.Contains(output, "| :--- | ---: | ---: | ---: | ---: |") {
					t.Errorf("missing alignment: %s", output)
				}
				// Check for protected paths with underscores
				if !strings.Contains(output, "| `src/user_service` | 3 | 10 | 5 | +5 |") {
					t.Errorf("missing entry 1: %s", output)
				}
				if !strings.Contains(output, "| `docs` | 1 | 0 | 10 | -10 |") {
					t.Errorf("missing entry 2: %s", output)
				}
				if !strings.Contains(output, "| **TOTAL** | **4** | **10** | **15** | **-5** |") {
					t.Errorf("missing total line: %s", output)
				}
				if strings.Contains(output, "Percent") || strings.Contains(output, "Graph") {
					t.Errorf("unexpected columns: %s", output)
				}
			},
		},
		{
			name:        "with percent",
			showPercent: true,
			showGraph:   false,
			checkFunc: func(t *testing.T, output string) {
				if !strings.Contains(output, "| Directory | Files | Added | Deleted | Net | Percent |") {
					t.Errorf("missing percent header: %s", output)
				}
				if !strings.Contains(output, "| `src/user_service` | 3 | 10 | 5 | +5 | 50.0% |") {
					t.Errorf("missing entry 1 with percent: %s", output)
				}
				if !strings.Contains(output, "| **TOTAL** | **4** | **10** | **15** | **-5** | **100.0%** |") {
					t.Errorf("missing total line with percent: %s", output)
				}
			},
		},
		{
			name:        "with graph",
			showPercent: false,
			showGraph:   true,
			checkFunc: func(t *testing.T, output string) {
				if !strings.Contains(output, "| Directory | Files | Added | Deleted | Net | Graph |") {
					t.Errorf("missing graph header: %s", output)
				}
				if !strings.Contains(output, "| :--- | ---: | ---: | ---: | ---: | :--- |") {
					t.Errorf("missing alignment for graph: %s", output)
				}
				// Total changes is 25, max length is 20.
				// For src/user_service: added 10, deleted 5.
				// Total graph bar will be `++++----` (scale depends on buildGraphBar)
				if !strings.Contains(output, "++++") {
					t.Errorf("missing graph string: %s", output)
				}
			},
		},
		{
			name:        "with stat (percent and graph)",
			showPercent: true,
			showGraph:   true,
			checkFunc: func(t *testing.T, output string) {
				if !strings.Contains(output, "| Directory | Files | Added | Deleted | Net | Percent | Graph |") {
					t.Errorf("missing stat headers: %s", output)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewMarkdownFormatterWithOptions(tt.showPercent, tt.showGraph)
			var buf bytes.Buffer
			err := f.Format(&buf, report)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			tt.checkFunc(t, buf.String())
		})
	}
}

func TestMarkdownFormatter_EmptyReport(t *testing.T) {
	report := &model.Report{
		Target:  ".",
		Depth:   1,
		Entries: []model.Entry{},
		Summary: model.Summary{
			TotalFiles:   0,
			TotalAdded:   0,
			TotalDeleted: 0,
			Net:          0,
		},
	}

	f := NewMarkdownFormatterWithOptions(true, true)
	var buf bytes.Buffer
	err := f.Format(&buf, report)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "| **TOTAL** | **0** | **0** | **0** | **0** | **0.0%** |  |") {
		t.Errorf("missing or incorrect empty total line: %s", output)
	}
}

func TestNewMarkdownFormatter(t *testing.T) {
	f := NewMarkdownFormatter()
	if f.MaxGraphLen != 20 {
		t.Errorf("expected MaxGraphLen 20, got %d", f.MaxGraphLen)
	}
	if f.ShowPercent || f.ShowGraph {
		t.Errorf("expected false for show percent and graph")
	}

	// Test zero maxGraphLen fallback
	f.MaxGraphLen = 0
	report := &model.Report{
		Target:  ".",
		Depth:   1,
		Entries: []model.Entry{},
	}
	var buf bytes.Buffer
	_ = f.Format(&buf, report) // triggers fallback to 20
}

type errWriter struct{}

func (e *errWriter) Write(p []byte) (n int, err error) {
	return 0, fmt.Errorf("simulated write error")
}

func TestMarkdownFormatter_WriteError(t *testing.T) {
	f := NewMarkdownFormatter()
	report := &model.Report{
		Target:  ".",
		Depth:   1,
		Entries: []model.Entry{},
	}
	err := f.Format(&errWriter{}, report)
	if err == nil {
		t.Errorf("expected error from failing writer, got nil")
	}
}

func TestMarkdownFormatter_BacktickEscaping(t *testing.T) {
	report := &model.Report{
		Target: "src/`target`/",
		Depth:  1,
		Entries: []model.Entry{
			{Path: "src/`bad_dir`/", Files: 1, Added: 5, Deleted: 2, Net: 3},
		},
		Summary: model.Summary{
			TotalFiles:   1,
			TotalAdded:   5,
			TotalDeleted: 2,
			Net:          3,
		},
	}

	f := NewMarkdownFormatter()
	var buf bytes.Buffer
	if err := f.Format(&buf, report); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	if strings.Contains(out, "```") {
		t.Errorf("unexpected triple backticks in output: %s", out)
	}
	if !strings.Contains(out, "`src/bad_dir/`") {
		t.Errorf("expected backticks to be stripped from inside path, got: %s", out)
	}
}
