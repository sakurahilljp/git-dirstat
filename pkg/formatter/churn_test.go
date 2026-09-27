package formatter

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/sakurahilljp/git-dirstat/pkg/model"
)

func sampleChurnReport() *model.ChurnReport {
	return &model.ChurnReport{
		Target: ".",
		Depth:  1,
		Summary: model.ChurnSummary{
			TotalCommits: 10,
			TotalFiles:   5,
			TotalAdded:   100,
			TotalDeleted: 20,
			TotalChurn:   120,
		},
		Entries: []model.ChurnEntry{
			{
				Path:    "pkg/",
				IsRoot:  false,
				Commits: 8,
				Files:   3,
				Added:   80,
				Deleted: 15,
				Churn:   95,
				Percent: 79.2,
			},
			{
				Path:    "cmd/",
				IsRoot:  false,
				Commits: 4,
				Files:   2,
				Added:   20,
				Deleted: 5,
				Churn:   25,
				Percent: 20.8,
			},
		},
	}
}

func TestChurnTableFormatter(t *testing.T) {
	rep := sampleChurnReport()
	f := NewChurnTableFormatter(true, true, true)
	var buf bytes.Buffer
	err := f.FormatChurn(&buf, rep)
	if err != nil {
		t.Fatalf("FormatChurn failed: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Directory") || !strings.Contains(out, "Commits") || !strings.Contains(out, "Churn") {
		t.Errorf("output missing expected headers: %s", out)
	}
	if !strings.Contains(out, "pkg/") || !strings.Contains(out, "cmd/") {
		t.Errorf("output missing directory rows: %s", out)
	}
	if !strings.Contains(out, "TOTAL") {
		t.Errorf("output missing TOTAL row: %s", out)
	}
}

func TestChurnMarkdownFormatter(t *testing.T) {
	rep := sampleChurnReport()
	f := NewChurnMarkdownFormatterWithOptions(true, true)
	var buf bytes.Buffer
	err := f.FormatChurn(&buf, rep)
	if err != nil {
		t.Fatalf("FormatChurn failed: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "| Directory | Commits | Files | Added | Deleted | Churn | Percent | Graph |") {
		t.Errorf("markdown header mismatch: %s", out)
	}
	if !strings.Contains(out, "`pkg/`") {
		t.Errorf("markdown row missing pkg/: %s", out)
	}
}

func TestChurnJSONFormatter(t *testing.T) {
	rep := sampleChurnReport()
	f := NewChurnJSONFormatter()
	var buf bytes.Buffer
	err := f.FormatChurn(&buf, rep)
	if err != nil {
		t.Fatalf("FormatChurn failed: %v", err)
	}

	var parsed model.ChurnReport
	err = json.Unmarshal(buf.Bytes(), &parsed)
	if err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}
	if parsed.Summary.TotalCommits != 10 {
		t.Errorf("expected 10 commits, got %d", parsed.Summary.TotalCommits)
	}
}

func TestChurnDelimitedFormatter(t *testing.T) {
	rep := sampleChurnReport()
	csvFmt := NewChurnCSVFormatterWithPercent(true)
	var buf bytes.Buffer
	err := csvFmt.FormatChurn(&buf, rep)
	if err != nil {
		t.Fatalf("FormatChurn CSV failed: %v", err)
	}

	csvOut := buf.String()
	if !strings.Contains(csvOut, "path,commits,files,added,deleted,churn,percent") {
		t.Errorf("unexpected CSV header: %s", csvOut)
	}

	tsvFmt := NewChurnTSVFormatterWithPercent(false)
	buf.Reset()
	err = tsvFmt.FormatChurn(&buf, rep)
	if err != nil {
		t.Fatalf("FormatChurn TSV failed: %v", err)
	}

	tsvOut := buf.String()
	if !strings.Contains(tsvOut, "path\tcommits\tfiles\tadded\tdeleted\tchurn") {
		t.Errorf("unexpected TSV header: %s", tsvOut)
	}
}
