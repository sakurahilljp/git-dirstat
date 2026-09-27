package aggregator

import (
	"testing"

	"github.com/sakurahilljp/git-dirstat/pkg/model"
)

func TestChurnAggregator_Basic(t *testing.T) {
	agg, err := NewChurnAggregator(ChurnAggregatorOptions{
		RepoRoot:   "/repo",
		Cwd:        "/repo",
		TargetPath: ".",
		Depth:      1,
		Sort:       model.SortCommits,
	})
	if err != nil {
		t.Fatalf("failed to create ChurnAggregator: %v", err)
	}

	// Commit 1 touches two files in pkg/
	err = agg.ConsumeCommit("c1", []model.FileDiff{
		{Path: "pkg/a.go", Added: 10, Deleted: 2},
		{Path: "pkg/b.go", Added: 5, Deleted: 1},
	})
	if err != nil {
		t.Fatalf("ConsumeCommit failed: %v", err)
	}

	// Commit 2 touches pkg/a.go and cmd/main.go
	err = agg.ConsumeCommit("c2", []model.FileDiff{
		{Path: "pkg/a.go", Added: 20, Deleted: 5},
		{Path: "cmd/main.go", Added: 50, Deleted: 10},
	})
	if err != nil {
		t.Fatalf("ConsumeCommit failed: %v", err)
	}

	// Commit 3 touches root README.md
	err = agg.ConsumeCommit("c3", []model.FileDiff{
		{Path: "README.md", Added: 5, Deleted: 0},
	})
	if err != nil {
		t.Fatalf("ConsumeCommit failed: %v", err)
	}

	report, err := agg.Result()
	if err != nil {
		t.Fatalf("Result failed: %v", err)
	}

	if report.Summary.TotalCommits != 3 {
		t.Errorf("expected 3 total commits, got %d", report.Summary.TotalCommits)
	}
	if report.Summary.TotalFiles != 4 {
		t.Errorf("expected 4 total unique files, got %d", report.Summary.TotalFiles)
	}
	if report.Summary.TotalAdded != 90 {
		t.Errorf("expected 90 total added, got %d", report.Summary.TotalAdded)
	}
	if report.Summary.TotalDeleted != 18 {
		t.Errorf("expected 18 total deleted, got %d", report.Summary.TotalDeleted)
	}
	if report.Summary.TotalChurn != 108 {
		t.Errorf("expected 108 total churn, got %d", report.Summary.TotalChurn)
	}

	// Verify entries: sorted by commits descending
	// pkg/ was in c1 and c2 -> 2 commits, 2 unique files (a.go, b.go), added 35, deleted 8, churn 43
	if len(report.Entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(report.Entries))
	}

	pkgEntry := report.Entries[0]
	if pkgEntry.Path != "pkg/" {
		t.Errorf("expected first entry to be pkg/, got %s", pkgEntry.Path)
	}
	if pkgEntry.Commits != 2 {
		t.Errorf("expected pkg/ commits=2, got %d", pkgEntry.Commits)
	}
	if pkgEntry.Files != 2 {
		t.Errorf("expected pkg/ files=2, got %d", pkgEntry.Files)
	}
	if pkgEntry.Added != 35 {
		t.Errorf("expected pkg/ added=35, got %d", pkgEntry.Added)
	}
	if pkgEntry.Deleted != 8 {
		t.Errorf("expected pkg/ deleted=8, got %d", pkgEntry.Deleted)
	}
	if pkgEntry.Churn != 43 {
		t.Errorf("expected pkg/ churn=43, got %d", pkgEntry.Churn)
	}
}

func TestChurnAggregator_TopAndReverse(t *testing.T) {
	agg, err := NewChurnAggregator(ChurnAggregatorOptions{
		RepoRoot:   "/repo",
		Cwd:        "/repo",
		TargetPath: ".",
		Depth:      1,
		Sort:       model.SortChurn,
		Top:        1,
	})
	if err != nil {
		t.Fatalf("failed to create ChurnAggregator: %v", err)
	}

	_ = agg.ConsumeCommit("c1", []model.FileDiff{
		{Path: "dirA/f.go", Added: 10, Deleted: 0}, // churn 10
		{Path: "dirB/f.go", Added: 50, Deleted: 0}, // churn 50
	})

	report, err := agg.Result()
	if err != nil {
		t.Fatalf("Result failed: %v", err)
	}

	if len(report.Entries) != 1 {
		t.Fatalf("expected 1 entry due to Top=1, got %d", len(report.Entries))
	}
	if report.Entries[0].Path != "dirB/" {
		t.Errorf("expected dirB/ with highest churn, got %s", report.Entries[0].Path)
	}
}

func TestChurnAggregator_Exclude(t *testing.T) {
	agg, err := NewChurnAggregator(ChurnAggregatorOptions{
		RepoRoot:   "/repo",
		Cwd:        "/repo",
		TargetPath: ".",
		Depth:      1,
		Exclude:    []string{"vendor/**"},
	})
	if err != nil {
		t.Fatalf("failed to create ChurnAggregator: %v", err)
	}

	_ = agg.ConsumeCommit("c1", []model.FileDiff{
		{Path: "vendor/lib.go", Added: 100, Deleted: 0},
		{Path: "src/main.go", Added: 10, Deleted: 0},
	})

	report, err := agg.Result()
	if err != nil {
		t.Fatalf("Result failed: %v", err)
	}

	if len(report.Entries) != 1 {
		t.Fatalf("expected 1 entry (vendor excluded), got %d", len(report.Entries))
	}
	if report.Entries[0].Path != "src/" {
		t.Errorf("expected src/, got %s", report.Entries[0].Path)
	}
}

func TestSortChurnEntries(t *testing.T) {
	entries := []model.ChurnEntry{
		{Path: "a/", Commits: 1, Churn: 10, Files: 1, Added: 5, Deleted: 5, Percent: 10.0},
		{Path: "b/", Commits: 2, Churn: 20, Files: 2, Added: 10, Deleted: 10, Percent: 20.0},
		{Path: "c/", Commits: 2, Churn: 10, Files: 3, Added: 5, Deleted: 5, Percent: 10.0},
		{Path: "d/", Commits: 3, Churn: 20, Files: 1, Added: 10, Deleted: 10, Percent: 20.0},
	}

	tests := []struct {
		sortField string
		expected  []string
	}{
		{model.SortPath, []string{"a/", "b/", "c/", "d/"}},
		{model.SortCommits, []string{"d/", "b/", "c/", "a/"}},
		{model.SortChurn, []string{"d/", "b/", "c/", "a/"}},
		{model.SortFiles, []string{"c/", "b/", "a/", "d/"}},
		{model.SortAdded, []string{"b/", "d/", "a/", "c/"}},
		{model.SortDeleted, []string{"b/", "d/", "a/", "c/"}},
		{model.SortPercent, []string{"b/", "d/", "a/", "c/"}},
		{"unknown", []string{"d/", "b/", "c/", "a/"}},
	}

	for _, tt := range tests {
		t.Run(tt.sortField, func(t *testing.T) {
			// Copy entries
			copied := make([]model.ChurnEntry, len(entries))
			copy(copied, entries)

			sortChurnEntries(copied, tt.sortField)

			var result []string
			for _, e := range copied {
				result = append(result, e.Path)
			}

			for i, r := range result {
				if r != tt.expected[i] {
					t.Errorf("expected %v, got %v", tt.expected, result)
					break
				}
			}
		})
	}
}

func TestChurnAggregator_PathFilter(t *testing.T) {
	agg, _ := NewChurnAggregator(ChurnAggregatorOptions{
		RepoRoot:   "/repo",
		Cwd:        "/repo",
		TargetPath: ".",
		Depth:      1,
		Exclude:    []string{"vendor/**"},
	})
	if agg.PathFilter() == nil {
		t.Error("expected PathFilter to not be nil")
	}
}
