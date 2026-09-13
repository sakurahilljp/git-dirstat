package aggregator

import (
	"reflect"
	"testing"

	"github.com/sakurahilljp/git-dirstat/pkg/model"
)

func TestAggregator(t *testing.T) {
	diffs := []model.FileDiff{
		{Path: "src/components/button/index.tsx", Added: 100, Deleted: 20},
		{Path: "src/components/button/button.test.tsx", Added: 50, Deleted: 5},
		{Path: "src/components/modal/index.tsx", Added: 80, Deleted: 10},
		{Path: "src/services/api.ts", Added: 200, Deleted: 50},
		{Path: "src/index.ts", Added: 30, Deleted: 5},                 // root file under src/
		{Path: "README.md", Added: 10, Deleted: 2},                    // outside src/
		{Path: "src/vendor/lib.js", Added: 1000, Deleted: 0},          // excluded
		{Path: "src/image.png", Added: 0, Deleted: 0, IsBinary: true}, // binary root file under src/
	}

	opts := AggregatorOptions{
		RepoRoot:   "/repo",
		Cwd:        "/repo",
		TargetPath: "src/",
		Depth:      1,
		Sort:       model.SortAdded,
		Reverse:    false,
		Exclude:    []string{"vendor/**", "src/vendor/**"},
	}

	report, err := Aggregate(diffs, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if report.Target != "src/" {
		t.Errorf("expected Target 'src/', got %q", report.Target)
	}
	if report.Depth != 1 {
		t.Errorf("expected Depth 1, got %d", report.Depth)
	}

	// Expected entries under src/ with depth 1:
	// - src/components/ (button/index.tsx, button/button.test.tsx, modal/index.tsx): 3 files, 230 added, 35 deleted, net 195
	// - src/services/ (api.ts): 1 file, 200 added, 50 deleted, net 150
	// - src/ (index.ts, image.png): 2 files, 30 added, 5 deleted, net 25, is_root: true
	// Sorted by Added descending:
	// 1. src/components/ (added: 230)
	// 2. src/services/ (added: 200)
	// 3. src/ (added: 30)

	if len(report.Entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(report.Entries))
	}

	e0 := report.Entries[0]
	if e0.Path != "src/components/" || e0.Files != 3 || e0.Added != 230 || e0.Deleted != 35 || e0.Net != 195 || e0.IsRoot {
		t.Errorf("unexpected entry 0: %+v", e0)
	}

	e1 := report.Entries[1]
	if e1.Path != "src/services/" || e1.Files != 1 || e1.Added != 200 || e1.Deleted != 50 || e1.Net != 150 || e1.IsRoot {
		t.Errorf("unexpected entry 1: %+v", e1)
	}

	e2 := report.Entries[2]
	if e2.Path != "src/" || e2.Files != 2 || e2.Added != 30 || e2.Deleted != 5 || e2.Net != 25 || !e2.IsRoot {
		t.Errorf("unexpected entry 2: %+v", e2)
	}

	// Total summary check:
	// Total files: 3 components + 1 service + 2 root = 6 files
	// Total added: 230 + 200 + 30 = 460
	// Total deleted: 35 + 50 + 5 = 90
	// Net: 370
	if report.Summary.TotalFiles != 6 || report.Summary.TotalAdded != 460 || report.Summary.TotalDeleted != 90 || report.Summary.Net != 370 {
		t.Errorf("unexpected summary: %+v", report.Summary)
	}
}

func TestDepth2(t *testing.T) {
	diffs := []model.FileDiff{
		{Path: "src/components/button/index.tsx", Added: 100, Deleted: 20},
		{Path: "src/components/modal/index.tsx", Added: 80, Deleted: 10},
		{Path: "src/services/api.ts", Added: 200, Deleted: 50},
	}

	opts := AggregatorOptions{
		RepoRoot:   "/repo",
		Cwd:        "/repo",
		TargetPath: "src/",
		Depth:      2,
		Sort:       model.SortAdded,
	}

	report, err := Aggregate(diffs, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// With Depth 2:
	// - src/services/ (fewer than depth 2 directory segments) -> src/services/ (added: 200)
	// - src/components/button/ -> (added: 100)
	// - src/components/modal/ -> (added: 80)
	if len(report.Entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(report.Entries))
	}
	if report.Entries[0].Path != "src/services/" {
		t.Errorf("expected src/services/, got %s", report.Entries[0].Path)
	}
	if report.Entries[1].Path != "src/components/button/" {
		t.Errorf("expected src/components/button/, got %s", report.Entries[1].Path)
	}
	if report.Entries[2].Path != "src/components/modal/" {
		t.Errorf("expected src/components/modal/, got %s", report.Entries[2].Path)
	}
}

func TestSortingAndTieBreak(t *testing.T) {
	entries := []model.Entry{
		{Path: "b/", Added: 100, Net: -10, Files: 2},
		{Path: "a/", Added: 100, Net: 20, Files: 5},
		{Path: "c/", Added: 50, Net: 50, Files: 1},
	}

	// Sort by Added: equal added for b/ and a/ -> tie-break path ascending -> a/ then b/
	sortEntries(entries, model.SortAdded)
	paths := []string{entries[0].Path, entries[1].Path, entries[2].Path}
	if !reflect.DeepEqual(paths, []string{"a/", "b/", "c/"}) {
		t.Errorf("SortAdded failed: got %v", paths)
	}

	// Sort by Net signed descending: c/ (50), a/ (20), b/ (-10)
	sortEntries(entries, model.SortNet)
	paths = []string{entries[0].Path, entries[1].Path, entries[2].Path}
	if !reflect.DeepEqual(paths, []string{"c/", "a/", "b/"}) {
		t.Errorf("SortNet failed: got %v", paths)
	}

	// Sort by Path ascending: a/, b/, c/
	sortEntries(entries, model.SortPath)
	paths = []string{entries[0].Path, entries[1].Path, entries[2].Path}
	if !reflect.DeepEqual(paths, []string{"a/", "b/", "c/"}) {
		t.Errorf("SortPath failed: got %v", paths)
	}
}

func TestEmptyDiffs(t *testing.T) {
	opts := AggregatorOptions{
		RepoRoot:   "/repo",
		Cwd:        "/repo",
		TargetPath: ".",
		Depth:      1,
		Sort:       model.SortAdded,
	}

	report, err := Aggregate([]model.FileDiff{}, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if report.Target != "." {
		t.Errorf("expected target '.', got %q", report.Target)
	}
	if report.Summary.TotalFiles != 0 || report.Summary.TotalAdded != 0 || report.Summary.TotalDeleted != 0 || report.Summary.Net != 0 {
		t.Errorf("expected 0 summary, got %+v", report.Summary)
	}
	if len(report.Entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(report.Entries))
	}
}

func TestNormalizeTarget_OutsideRepo(t *testing.T) {
	_, _, err := NormalizeTarget("/repo", "/repo", "../outside")
	if err == nil {
		t.Fatalf("expected error for target outside repo")
	}
	if exitErr, ok := err.(*model.ExitCodeError); !ok || exitErr.Code != 2 {
		t.Errorf("expected ExitCode 2, got %v", err)
	}
}

func TestSortingMoreFields(t *testing.T) {
	entries := []model.Entry{
		{Path: "b/", Files: 10, Deleted: 20},
		{Path: "a/", Files: 10, Deleted: 50},
		{Path: "c/", Files: 5, Deleted: 50},
	}

	// Sort by Files descending, tie-breaker path ascending
	sortEntries(entries, model.SortFiles)
	if entries[0].Path != "a/" || entries[1].Path != "b/" || entries[2].Path != "c/" {
		t.Errorf("SortFiles failed: %v", entries)
	}

	// Sort by Deleted descending, tie-breaker path ascending
	sortEntries(entries, model.SortDeleted)
	if entries[0].Path != "a/" || entries[1].Path != "c/" || entries[2].Path != "b/" {
		t.Errorf("SortDeleted failed: %v", entries)
	}
}

func TestStreamAggregatorDirect(t *testing.T) {
	opts := AggregatorOptions{
		RepoRoot:   "/repo",
		Cwd:        "/repo",
		TargetPath: "pkg/",
		Depth:      1,
		Sort:       model.SortAdded,
	}

	agg, err := NewStreamAggregator(opts)
	if err != nil {
		t.Fatalf("failed to create StreamAggregator: %v", err)
	}

	pf := agg.PathFilter()
	if pf == nil || pf.TargetPrefix() != "pkg/" {
		t.Fatalf("expected PathFilter with prefix 'pkg/', got %v", pf)
	}

	diffs := []model.FileDiff{
		{Path: "pkg/a/file1.go", Added: 10, Deleted: 2},
		{Path: "pkg/b/file2.go", Added: 20, Deleted: 5},
		{Path: "pkg/root.go", Added: 5, Deleted: 1},
	}

	for _, d := range diffs {
		if err := agg.Consume(d); err != nil {
			t.Fatalf("Consume failed: %v", err)
		}
	}

	report, err := agg.Result()
	if err != nil {
		t.Fatalf("Result failed: %v", err)
	}

	if report.Summary.TotalFiles != 3 {
		t.Errorf("expected 3 total files, got %d", report.Summary.TotalFiles)
	}
	if report.Summary.TotalAdded != 35 {
		t.Errorf("expected 35 total added, got %d", report.Summary.TotalAdded)
	}
	if len(report.Entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(report.Entries))
	}
}
