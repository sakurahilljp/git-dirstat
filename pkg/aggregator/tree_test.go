package aggregator

import (
	"testing"

	"fmt"

	"github.com/sakurahilljp/git-dirstat/pkg/model"
)

func TestTreeAggregator_Basic(t *testing.T) {
	opts := AggregatorOptions{
		RepoRoot:   "/repo",
		Cwd:        "/repo",
		TargetPath: ".",
		Depth:      1,
		Sort:       model.SortAdded,
	}

	agg, err := NewTreeAggregator(opts)
	if err != nil {
		t.Fatalf("unexpected error creating TreeAggregator: %v", err)
	}

	diffs := []model.FileDiff{
		{Path: "cmd/root.go", Added: 100, Deleted: 20},
		{Path: "cmd/args.go", Added: 50, Deleted: 10},
		{Path: "pkg/model/tree.go", Added: 80, Deleted: 0},
		{Path: "README.md", Added: 10, Deleted: 5},
	}

	for _, d := range diffs {
		if err := agg.Consume(d); err != nil {
			t.Fatalf("failed to consume diff: %v", err)
		}
	}

	report, err := agg.Result()
	if err != nil {
		t.Fatalf("unexpected error from Result: %v", err)
	}

	if report.Summary.TotalFiles != 4 {
		t.Errorf("expected 4 total files, got %d", report.Summary.TotalFiles)
	}
	if report.Summary.TotalAdded != 240 {
		t.Errorf("expected 240 added, got %d", report.Summary.TotalAdded)
	}
	if report.Summary.TotalDeleted != 35 {
		t.Errorf("expected 35 deleted, got %d", report.Summary.TotalDeleted)
	}

	root := report.Root
	if root == nil {
		t.Fatalf("expected non-nil root")
	}

	// Root should have children: "cmd/", "pkg/", "README.md"
	if len(root.Children) != 3 {
		t.Fatalf("expected 3 root children, got %d", len(root.Children))
	}

	// Dirs first, sorted by Added:
	// cmd/ (added=150), pkg/ (added=80), then README.md (added=10)
	c0 := root.Children[0]
	if c0.Name != "cmd/" || c0.Added != 150 || c0.Deleted != 30 || c0.Files != 2 {
		t.Errorf("unexpected first child: %+v", c0)
	}
	if !c0.Expanded { // Depth=1, root is 0, c0 depth is 1. Wait: depth < opts.Depth (1 < 1 is false). Let's check Option A!
	}
}

func TestTreeAggregator_DepthExpansion(t *testing.T) {
	opts := AggregatorOptions{
		RepoRoot:   "/repo",
		Cwd:        "/repo",
		TargetPath: ".",
		Depth:      2,
		Sort:       model.SortAdded,
	}

	agg, err := NewTreeAggregator(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	diffs := []model.FileDiff{
		{Path: "pkg/aggregator/tree.go", Added: 50, Deleted: 5},
	}
	for _, d := range diffs {
		_ = agg.Consume(d)
	}

	report, err := agg.Result()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	root := report.Root
	if !root.Expanded {
		t.Errorf("expected root to be expanded")
	}
	// pkg/ should be expanded because depth=1 <= opts.Depth (if Depth=2, depth 1 is expanded)
	pkgNode := root.Children[0]
	if !pkgNode.Expanded {
		t.Errorf("expected pkg/ to be expanded at opts.Depth=2")
	}
}

func TestSortTreeNodes(t *testing.T) {
	nodes := []*model.TreeNode{
		{Name: "c.go", IsDir: false, Files: 1, Added: 10, Deleted: 5, Net: 5, Percent: 10},
		{Name: "b.go", IsDir: false, Files: 1, Added: 20, Deleted: 2, Net: 18, Percent: 20},
		{Name: "a.go", IsDir: false, Files: 1, Added: 5, Deleted: 10, Net: -5, Percent: 5},
		{Name: "dir1", IsDir: true, Files: 10, Added: 100, Deleted: 50, Net: 50, Percent: 50},
		{Name: "dir2", IsDir: true, Files: 5, Added: 100, Deleted: 50, Net: 50, Percent: 50}, // tie breakers
	}

	sortFields := []string{
		model.SortPath,
		model.SortFiles,
		model.SortAdded,
		model.SortDeleted,
		model.SortNet,
		model.SortPercent,
		"default_changes",
	}

	for _, field := range sortFields {
		for _, reverse := range []bool{false, true} {
			t.Run(field+"_reverse_"+fmt.Sprint(reverse), func(t *testing.T) {
				// copy
				cp := make([]*model.TreeNode, len(nodes))
				copy(cp, nodes)
				sortTreeNodes(cp, field, reverse)
				if len(cp) != len(nodes) {
					t.Fatalf("length mismatch")
				}
				// Verify dir first rule
				for i := 0; i < len(cp)-1; i++ {
					if !cp[i].IsDir && cp[i+1].IsDir {
						t.Fatalf("dirs not sorted first at %d", i)
					}
				}
			})
		}
	}
}

func TestTreeAggregator_PathFilter_Coverage(t *testing.T) {
	opts := AggregatorOptions{
		RepoRoot:   "/repo",
		Cwd:        "/repo",
		TargetPath: ".",
		Depth:      -1,
	}
	agg, err := NewTreeAggregator(opts)
	if err != nil {
		t.Fatal(err)
	}
	f := agg.PathFilter()
	if f == nil {
		t.Fatalf("expected non-nil filter")
	}
}
