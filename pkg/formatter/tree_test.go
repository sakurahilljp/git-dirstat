package formatter

import (
	"bytes"
	"strings"
	"testing"

	"github.com/sakurahilljp/git-dirstat/pkg/model"
)

func TestTreeFormatter_Basic(t *testing.T) {
	rootNode := &model.TreeNode{
		Name:    ".",
		Path:    ".",
		IsDir:   true,
		Files:   3,
		Added:   150,
		Deleted: 50,
		Net:     100,
		Percent: 100.0,
		Depth:   0,
		Children: []*model.TreeNode{
			{
				Name:    "cmd/",
				Path:    "cmd/",
				IsDir:   true,
				Files:   2,
				Added:   100,
				Deleted: 20,
				Net:     80,
				Percent: 60.0,
				Depth:   1,
				Children: []*model.TreeNode{
					{
						Name:    "root.go",
						Path:    "cmd/root.go",
						IsDir:   false,
						Files:   1,
						Added:   100,
						Deleted: 20,
						Net:     80,
						Percent: 60.0,
						Depth:   2,
					},
				},
			},
			{
				Name:    "README.md",
				Path:    "README.md",
				IsDir:   false,
				Files:   1,
				Added:   50,
				Deleted: 30,
				Net:     20,
				Percent: 40.0,
				Depth:   1,
			},
		},
	}

	report := &model.TreeReport{
		Target: ".",
		Summary: model.Summary{
			TotalFiles:   3,
			TotalAdded:   150,
			TotalDeleted: 50,
			Net:          100,
		},
		Root: rootNode,
	}

	tf := NewTreeFormatterWithOptions(true, true, true, 0)
	var buf bytes.Buffer
	err := tf.Format(&buf, report)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Target: .") {
		t.Errorf("missing target header in output:\n%s", out)
	}
	if !strings.Contains(out, "├── cmd/") {
		t.Errorf("missing tree branch for cmd/:\n%s", out)
	}
	if !strings.Contains(out, "│   └── root.go") {
		t.Errorf("missing nested tree branch for root.go:\n%s", out)
	}
	if !strings.Contains(out, "└── README.md") {
		t.Errorf("missing tree branch for README.md:\n%s", out)
	}
	if !strings.Contains(out, "60.0%") {
		t.Errorf("missing percent in output:\n%s", out)
	}
}

func TestTreeFormatter_MaxDepth(t *testing.T) {
	rootNode := &model.TreeNode{
		Name:  ".",
		Path:  ".",
		IsDir: true,
		Depth: 0,
		Children: []*model.TreeNode{
			{
				Name:  "cmd/",
				Path:  "cmd/",
				IsDir: true,
				Depth: 1,
				Children: []*model.TreeNode{
					{
						Name:  "root.go",
						Path:  "cmd/root.go",
						IsDir: false,
						Depth: 2,
					},
				},
			},
		},
	}

	report := &model.TreeReport{
		Target: ".",
		Root:   rootNode,
	}

	// MaxDepth = 1: root.go should NOT be rendered
	tf := NewTreeFormatterWithOptions(true, false, false, 1)
	var buf bytes.Buffer
	err := tf.Format(&buf, report)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "└── cmd/") {
		t.Errorf("expected cmd/ to be in output:\n%s", out)
	}
	if strings.Contains(out, "root.go") {
		t.Errorf("root.go should be truncated by MaxDepth=1, but found in output:\n%s", out)
	}
}
