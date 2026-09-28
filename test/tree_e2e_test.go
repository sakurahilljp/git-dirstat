package test

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/sakurahilljp/git-dirstat/cmd"
)

func TestE2E_TreeView(t *testing.T) {
	dir, repo := setupTestRepo(t)
	defer os.RemoveAll(dir)

	origWd, _ := os.Getwd()
	defer os.Chdir(origWd)
	_ = os.Chdir(dir)

	// Commit initial files
	c1 := commitFile(t, repo, dir, "src/components/button.go", "package components\n// button line 1\n// button line 2\n", "Initial button")
	commitFile(t, repo, dir, "src/components/modal.go", "package components\n// modal line 1\n", "Initial modal")
	c2 := commitFile(t, repo, dir, "docs/readme.md", "# Docs\nDocumentation text\n", "Initial docs")

	t.Run("tree_flag_output", func(t *testing.T) {
		rootCmd := cmd.NewRootCommand()
		var outBuf bytes.Buffer
		rootCmd.SetOut(&outBuf)
		rootCmd.SetArgs([]string{c1.String() + ".." + c2.String(), "--tree"})

		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("unexpected error executing root command with --tree: %v", err)
		}

		out := outBuf.String()
		if !strings.Contains(out, "Target: .") {
			t.Errorf("missing Target header in tree output:\n%s", out)
		}
		if !strings.Contains(out, "src/") {
			t.Errorf("missing src/ in tree output:\n%s", out)
		}
		if !strings.Contains(out, "docs/") {
			t.Errorf("missing docs/ in tree output:\n%s", out)
		}
		if !strings.Contains(out, "├── ") && !strings.Contains(out, "└── ") {
			t.Errorf("missing tree branch symbols in output:\n%s", out)
		}
	})

	t.Run("format_tree_output_with_stat", func(t *testing.T) {
		rootCmd := cmd.NewRootCommand()
		var outBuf bytes.Buffer
		rootCmd.SetOut(&outBuf)
		rootCmd.SetArgs([]string{c1.String() + ".." + c2.String(), "-f", "tree", "--stat"})

		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		out := outBuf.String()
		if !strings.Contains(out, "Percent") || !strings.Contains(out, "Graph") {
			t.Errorf("expected Percent and Graph columns in output:\n%s", out)
		}
		if !strings.Contains(out, "%") {
			t.Errorf("expected percentage values in output:\n%s", out)
		}
	})

	t.Run("tree_with_depth_limit", func(t *testing.T) {
		rootCmd := cmd.NewRootCommand()
		var outBuf bytes.Buffer
		rootCmd.SetOut(&outBuf)
		rootCmd.SetArgs([]string{c1.String() + ".." + c2.String(), "--tree", "-d", "1"})

		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		out := outBuf.String()
		if !strings.Contains(out, "src/") {
			t.Errorf("expected src/ at depth 1:\n%s", out)
		}
		// components/ is depth 2, should NOT appear when depth=1
		if strings.Contains(out, "components/") {
			t.Errorf("components/ should be truncated by depth=1, but appeared in output:\n%s", out)
		}
	})

	t.Run("interactive_in_non_terminal_fails_gracefully", func(t *testing.T) {
		rootCmd := cmd.NewRootCommand()
		var outBuf bytes.Buffer
		rootCmd.SetOut(&outBuf)
		rootCmd.SetArgs([]string{"-i"})

		err := rootCmd.Execute()
		if err == nil {
			t.Fatalf("expected error when running -i in non-terminal test environment")
		}
		if !strings.Contains(err.Error(), "non-terminal") {
			t.Errorf("unexpected error message: %v", err)
		}
	})
}
