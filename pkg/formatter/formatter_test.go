package formatter

import (
	"bytes"
	"strings"
	"testing"

	"github.com/sakurahilljp/git-dirstat/pkg/model"
)

func sampleReport() *model.Report {
	return &model.Report{
		Target: "src/",
		Depth:  1,
		Summary: model.Summary{
			TotalFiles:   28,
			TotalAdded:   1420,
			TotalDeleted: 310,
			Net:          1110,
		},
		Entries: []model.Entry{
			{Path: "src/components/", Files: 15, Added: 820, Deleted: 150, Net: 670},
			{Path: "src/services/", Files: 8, Added: 450, Deleted: 120, Net: 330},
			{Path: "src/utils/", Files: 3, Added: 120, Deleted: 35, Net: 85},
			{Path: "src/", IsRoot: true, Files: 2, Added: 30, Deleted: 5, Net: 25},
		},
	}
}

func emptyReport() *model.Report {
	return &model.Report{
		Target: "src/",
		Depth:  1,
		Summary: model.Summary{
			TotalFiles:   0,
			TotalAdded:   0,
			TotalDeleted: 0,
			Net:          0,
		},
		Entries: []model.Entry{},
	}
}

func TestTableFormatter(t *testing.T) {
	tf := NewTableFormatter(true) // no color

	var buf bytes.Buffer
	if err := tf.Format(&buf, sampleReport()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Target: src/ (Depth: 1)") {
		t.Errorf("missing target header: %s", out)
	}
	if !strings.Contains(out, "src/components/") || !strings.Contains(out, "+670") {
		t.Errorf("missing component entry or net formatting: %s", out)
	}
	if !strings.Contains(out, "src/ (root files)") {
		t.Errorf("missing root files display: %s", out)
	}
	if !strings.Contains(out, "TOTAL") || !strings.Contains(out, "+1110") {
		t.Errorf("missing TOTAL row: %s", out)
	}

	// Test empty
	buf.Reset()
	if err := tf.Format(&buf, emptyReport()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	emptyOut := buf.String()
	if !strings.Contains(emptyOut, "TOTAL") {
		t.Errorf("empty report should still display TOTAL: %s", emptyOut)
	}
}

func TestJSONFormatter(t *testing.T) {
	jf := NewJSONFormatter()

	var buf bytes.Buffer
	if err := jf.Format(&buf, sampleReport()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, `"target": "src/"`) {
		t.Errorf("missing target in json: %s", out)
	}
	if !strings.Contains(out, `"is_root": true`) {
		t.Errorf("missing is_root in json: %s", out)
	}

	// Test empty
	buf.Reset()
	if err := jf.Format(&buf, emptyReport()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), `"entries": []`) {
		t.Errorf("empty entries should be []: %s", buf.String())
	}
}

func TestDelimitedFormatters(t *testing.T) {
	csvFmt := NewCSVFormatter()
	var buf bytes.Buffer
	if err := csvFmt.Format(&buf, sampleReport()); err != nil {
		t.Fatalf("unexpected csv error: %v", err)
	}

	csvOut := buf.String()
	lines := strings.Split(strings.TrimSpace(csvOut), "\n")
	if len(lines) != 5 { // 1 header + 4 data
		t.Errorf("expected 5 lines in CSV, got %d:\n%s", len(lines), csvOut)
	}
	if lines[0] != "path,files,added,deleted,net" {
		t.Errorf("unexpected csv header: %s", lines[0])
	}
	if lines[4] != "src/ (root files),2,30,5,25" {
		t.Errorf("unexpected csv root files line: %s", lines[4])
	}

	// Test TSV
	tsvFmt := NewTSVFormatter()
	buf.Reset()
	if err := tsvFmt.Format(&buf, sampleReport()); err != nil {
		t.Fatalf("unexpected tsv error: %v", err)
	}
	tsvLines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(tsvLines) != 5 {
		t.Errorf("expected 5 lines in TSV, got %d", len(tsvLines))
	}
	if tsvLines[0] != "path\tfiles\tadded\tdeleted\tnet" {
		t.Errorf("unexpected tsv header: %s", tsvLines[0])
	}

	// Test Empty CSV
	buf.Reset()
	if err := csvFmt.Format(&buf, emptyReport()); err != nil {
		t.Fatalf("unexpected empty csv error: %v", err)
	}
	emptyCsv := strings.TrimSpace(buf.String())
	if emptyCsv != "path,files,added,deleted,net" {
		t.Errorf("empty csv should only have header, got: %s", emptyCsv)
	}
}

func TestNegativeNetAndDotTarget(t *testing.T) {
	rep := &model.Report{
		Target: ".",
		Depth:  1,
		Summary: model.Summary{
			TotalFiles:   1,
			TotalAdded:   10,
			TotalDeleted: 30,
			Net:          -20,
		},
		Entries: []model.Entry{
			{Path: ".", IsRoot: true, Files: 1, Added: 10, Deleted: 30, Net: -20},
		},
	}

	// Table format: should have "(root files)" and "-20"
	tf := NewTableFormatter(true)
	var buf bytes.Buffer
	_ = tf.Format(&buf, rep)
	out := buf.String()
	if !strings.Contains(out, "(root files)") {
		t.Errorf("expected '(root files)' in table output, got: %s", out)
	}
	if !strings.Contains(out, "-20") {
		t.Errorf("expected '-20' in table output, got: %s", out)
	}

	// CSV format
	buf.Reset()
	csvFmt := NewCSVFormatter()
	_ = csvFmt.Format(&buf, rep)
	csvOut := buf.String()
	if !strings.Contains(csvOut, "(root files),1,10,30,-20") {
		t.Errorf("expected '(root files),1,10,30,-20' in CSV, got: %s", csvOut)
	}
}
