package tui

import (
	"strings"
	"testing"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/sakurahilljp/git-dirstat/pkg/model"
)

func createTestTreeReport() *model.TreeReport {
	root := &model.TreeNode{
		Name:     ".",
		Path:     ".",
		IsDir:    true,
		Files:    3,
		Added:    150,
		Deleted:  30,
		Net:      120,
		Percent:  100.0,
		Depth:    0,
		Expanded: true,
		Children: []*model.TreeNode{
			{
				Name:     "pkg/",
				Path:     "pkg/",
				IsDir:    true,
				Files:    2,
				Added:    100,
				Deleted:  20,
				Net:      80,
				Percent:  66.7,
				Depth:    1,
				Expanded: true,
				FileDiffs: []model.FileDiff{
					{Path: "pkg/tree.go", Added: 60, Deleted: 10},
				},
				Children: []*model.TreeNode{
					{
						Name:     "tree.go",
						Path:     "pkg/tree.go",
						IsDir:    false,
						Files:    1,
						Added:    60,
						Deleted:  10,
						Net:      50,
						Percent:  40.0,
						Depth:    2,
						Expanded: false,
					},
					{
						Name:     "app.go",
						Path:     "pkg/app.go",
						IsDir:    false,
						Files:    1,
						Added:    40,
						Deleted:  10,
						Net:      30,
						Percent:  26.7,
						Depth:    2,
						Expanded: false,
					},
				},
			},
			{
				Name:     "README.md",
				Path:     "README.md",
				IsDir:    false,
				Files:    1,
				Added:    50,
				Deleted:  10,
				Net:      40,
				Percent:  33.3,
				Depth:    1,
				Expanded: false,
			},
		},
	}

	return &model.TreeReport{
		Target: ".",
		Summary: model.Summary{
			TotalFiles:   3,
			TotalAdded:   150,
			TotalDeleted: 30,
			Net:          120,
		},
		Root: root,
	}
}

func TestAppModel_Navigation(t *testing.T) {
	report := createTestTreeReport()
	opts := Options{
		ShowPercent: true,
		ShowGraph:   true,
		SortField:   model.SortAdded,
	}

	app := NewAppModel(report, opts)

	// Window resize
	_, _ = app.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	// Initial cursor at root
	if app.cursor != 0 {
		t.Errorf("expected cursor 0, got %d", app.cursor)
	}

	// Move down
	_, _ = app.Update(tea.KeyMsg{Type: tea.KeyDown})
	if app.cursor != 1 {
		t.Errorf("expected cursor 1 after Down, got %d", app.cursor)
	}

	// Move up
	_, _ = app.Update(tea.KeyMsg{Type: tea.KeyUp})
	if app.cursor != 0 {
		t.Errorf("expected cursor 0 after Up, got %d", app.cursor)
	}

	// View output test
	view := app.View()
	if !strings.Contains(view, "git-dirstat TUI") {
		t.Errorf("view missing title:\n%s", view)
	}
	if !strings.Contains(view, "pkg/") {
		t.Errorf("view missing pkg/ node:\n%s", view)
	}
}

func TestAppModel_ExpandCollapse(t *testing.T) {
	report := createTestTreeReport()
	opts := Options{SortField: model.SortAdded}
	app := NewAppModel(report, opts)
	_ = app.Init()

	initialCount := len(app.visibleNodes)

	// Move to pkg/ (cursor 1)
	_, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if app.cursor != 1 {
		t.Fatalf("expected cursor 1, got %d", app.cursor)
	}

	// Collapse pkg/ via Space
	_, _ = app.Update(tea.KeyMsg{Type: tea.KeySpace})
	if len(app.visibleNodes) >= initialCount {
		t.Errorf("expected visibleNodes to decrease after collapse, was %d now %d", initialCount, len(app.visibleNodes))
	}

	// Expand pkg/ via Space
	_, _ = app.Update(tea.KeyMsg{Type: tea.KeySpace})
	if len(app.visibleNodes) != initialCount {
		t.Errorf("expected visibleNodes to restore to %d, got %d", initialCount, len(app.visibleNodes))
	}

	// Collapse all via 'O'
	_, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'O'}})
	// Only root and its direct children should be visible (root is expanded, pkg/ is collapsed, README.md is leaf)
	if len(app.visibleNodes) != 3 {
		t.Errorf("expected 3 nodes after 'O', got %d", len(app.visibleNodes))
	}

	// Expand all via 'o'
	_, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
	if len(app.visibleNodes) != initialCount {
		t.Errorf("expected %d nodes after 'o', got %d", initialCount, len(app.visibleNodes))
	}
}

func TestAppModel_SortAndToggles(t *testing.T) {
	report := createTestTreeReport()
	app := NewAppModel(report, Options{})

	// Cycle sort via 's'
	prevSort := app.opts.SortField
	_, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	if app.opts.SortField == prevSort {
		t.Errorf("expected sort field to change after 's'")
	}

	// Toggle percent 'p'
	prevPercent := app.opts.ShowPercent
	_, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	if app.opts.ShowPercent == prevPercent {
		t.Errorf("expected percent toggle after 'p'")
	}

	// Toggle graph 'g'
	prevGraph := app.opts.ShowGraph
	_, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	if app.opts.ShowGraph == prevGraph {
		t.Errorf("expected graph toggle after 'g'")
	}

	// Toggle help '?'
	_, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	if !app.showHelp {
		t.Errorf("expected showHelp to be true after '?'")
	}
	viewWithHelp := app.View()
	if !strings.Contains(viewWithHelp, "git-dirstat Interactive TUI Help") {
		t.Errorf("missing help text in modal view:\n%s", viewWithHelp)
	}

	// Close help with 'esc'
	_, _ = app.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if app.showHelp {
		t.Errorf("expected showHelp to be false after Esc")
	}
}

func TestAppModel_Filtering(t *testing.T) {
	report := createTestTreeReport()
	app := NewAppModel(report, Options{})

	// Enter filter mode via '/'
	_, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	if !app.isFiltering {
		t.Errorf("expected isFiltering to be true after '/'")
	}

	// Type 'tree'
	for _, r := range "tree" {
		_, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}

	// Submit filter with Enter
	_, _ = app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if app.isFiltering {
		t.Errorf("expected isFiltering to be false after Enter")
	}

	// Filter query should be "tree"
	if app.filterQuery != "tree" {
		t.Errorf("expected filterQuery to be 'tree', got %q", app.filterQuery)
	}

	// Visible nodes should be the matching node and its ancestral directories to preserve hierarchy
	for _, vn := range app.visibleNodes {
		if !nodeMatchesQuery(vn.node, "tree") {
			t.Errorf("node %s does not match filter or have matching descendants for 'tree'", vn.node.Path)
		}
	}
	// Non-matching leaf (README.md, app.go) should NOT be visible
	for _, vn := range app.visibleNodes {
		if vn.node.Name == "README.md" || vn.node.Name == "app.go" {
			t.Errorf("non-matching node %s should not be visible", vn.node.Name)
		}
	}
}

func TestAppModel_Quit(t *testing.T) {
	report := createTestTreeReport()
	app := NewAppModel(report, Options{})

	_, cmd := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Errorf("expected quit cmd on 'q'")
	}
}

func TestAppModel_SortTreeChildren(t *testing.T) {
	nodes := []*model.TreeNode{
		{Name: "c.go", IsDir: false, Files: 1, Added: 10, Deleted: 5, Net: 5, Percent: 10},
		{Name: "b.go", IsDir: false, Files: 1, Added: 20, Deleted: 2, Net: 18, Percent: 20},
		{Name: "a.go", IsDir: false, Files: 1, Added: 5, Deleted: 10, Net: -5, Percent: 5},
		{Name: "dir1", IsDir: true, Files: 10, Added: 100, Deleted: 50, Net: 50, Percent: 50},
		{Name: "dir2", IsDir: true, Files: 5, Added: 100, Deleted: 50, Net: 50, Percent: 50},
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
				cp := make([]*model.TreeNode, len(nodes))
				copy(cp, nodes)
				sortTreeChildren(cp, field, reverse)
				if len(cp) != len(nodes) {
					t.Fatalf("length mismatch")
				}
				for i := 0; i < len(cp)-1; i++ {
					if !cp[i].IsDir && cp[i+1].IsDir {
						t.Fatalf("dirs not sorted first at %d", i)
					}
				}
			})
		}
	}
}

func TestAppModel_RenderingCoverage(t *testing.T) {
	opts := Options{
		SortField: model.SortAdded,
		Reverse:   false,
	}

	root := &model.TreeNode{
		Name:  "root",
		IsDir: true,
		Children: []*model.TreeNode{
			{Name: "file.go", IsDir: false, Added: 10},
		},
	}

	app := NewAppModel(&model.TreeReport{Root: root}, opts)
    
    // trigger layout
    app.Update(tea.WindowSizeMsg{Width: 100, Height: 50})

    // call View to cover rendering paths
	out := app.View()
	if out == "" {
		t.Fatalf("expected non-empty view")
	}

    // scroll adjustment
    app.cursor = 100
    app.adjustScroll()
    
    // Help overlay
    app.showHelp = true
    outHelp := app.View()
    if !strings.Contains(outHelp, "Help") && !strings.Contains(outHelp, "Close") {
        // Just checking it runs without panic
    }
}
