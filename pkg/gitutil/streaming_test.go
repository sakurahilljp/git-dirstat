package gitutil

import (
	"os"
	"strings"
	"testing"

	"github.com/sakurahilljp/git-dirstat/pkg/filter"
	"github.com/sakurahilljp/git-dirstat/pkg/model"
)

func TestDiffCommitsStream_WithPreFilter(t *testing.T) {
	dir, repo := createTempGitRepo(t)
	defer os.RemoveAll(dir)

	h1 := addCommit(t, repo, dir, "src/main.go", []byte("package main\n\nfunc main() {}\n"), "c1")
	h2 := addCommit(t, repo, dir, "pkg/util.go", []byte("package util\n"), "c2")
	_ = addCommit(t, repo, dir, "src/vendor/lib.go", []byte("// vendor code\n"), "c3")
	h4 := addCommit(t, repo, dir, "src/main.go", []byte("package main\n\nfunc main() {\n\tprintln(\"hello\")\n}\n"), "c4")

	c1, err := repo.CommitObject(h1)
	if err != nil {
		t.Fatalf("failed to get c1: %v", err)
	}
	c4, err := repo.CommitObject(h4)
	if err != nil {
		t.Fatalf("failed to get c4: %v", err)
	}

	// Filter: only src/ target, exclude vendor/**
	pf := filter.NewPathFilter("src/", []string{"vendor/**", "src/vendor/**"})

	var streamedDiffs []model.FileDiff
	err = DiffCommitsStream(c1, c4, pf, func(d model.FileDiff) error {
		streamedDiffs = append(streamedDiffs, d)
		return nil
	})
	if err != nil {
		t.Fatalf("DiffCommitsStream failed: %v", err)
	}

	// Only src/main.go should be present.
	// pkg/util.go (outside target) and src/vendor/lib.go (excluded) must be skipped.
	if len(streamedDiffs) != 1 {
		t.Fatalf("expected 1 diff, got %d: %+v", len(streamedDiffs), streamedDiffs)
	}
	if streamedDiffs[0].Path != "src/main.go" {
		t.Errorf("expected src/main.go, got %s", streamedDiffs[0].Path)
	}

	_ = h2 // avoid unused variable
}

func TestCountLinesFromReader(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int
	}{
		{"empty", "", 0},
		{"single line with newline", "line1\n", 1},
		{"single line without newline", "line1", 1},
		{"multiple lines with newline", "line1\nline2\nline3\n", 3},
		{"multiple lines without trailing newline", "line1\nline2\nline3", 3},
		{"empty lines", "\n\n\n", 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := countLinesFromReader(strings.NewReader(tt.input))
			if err != nil {
				t.Fatalf("countLinesFromReader failed: %v", err)
			}
			if got != tt.want {
				t.Errorf("countLinesFromReader() = %d, want %d", got, tt.want)
			}
		})
	}
}
