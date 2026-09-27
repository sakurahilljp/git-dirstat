package test

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/sakurahilljp/git-dirstat/cmd"
	"github.com/sakurahilljp/git-dirstat/pkg/model"
)

func runChurnCmdInDir(dir string, args []string) (string, error) {
	origDir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	if err := os.Chdir(dir); err != nil {
		return "", err
	}
	defer os.Chdir(origDir)

	rootCmd := cmd.NewRootCommand()
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs(args)

	err = rootCmd.Execute()
	return buf.String(), err
}

func TestE2E_Churn(t *testing.T) {
	dir, repo := setupTestRepo(t)
	defer os.RemoveAll(dir)

	// Commit 1 (root): initial setup
	_ = commitFile(t, repo, dir, "src/main.go", "package main\n\nfunc main() {}\n", "initial commit")
	_ = commitFile(t, repo, dir, "docs/readme.md", "# Project\n", "add docs")

	// Commit 2: update src/main.go and add src/util.go
	_ = commitFile(t, repo, dir, "src/main.go", "package main\n\nfunc main() {\n\tprintln(\"run\")\n}\n", "update main")
	_ = commitFile(t, repo, dir, "src/util.go", "package main\n\nfunc helper() {}\n", "add util")

	// Commit 3: update src/main.go again
	_ = commitFile(t, repo, dir, "src/main.go", "package main\n\nfunc main() {\n\tprintln(\"v2\")\n}\n", "update main again")

	t.Run("basic churn table output", func(t *testing.T) {
		out, err := runChurnCmdInDir(dir, []string{"churn"})
		if err != nil {
			t.Fatalf("churn failed: %v\nOutput: %s", err, out)
		}
		if !strings.Contains(out, "Directory") || !strings.Contains(out, "Commits") || !strings.Contains(out, "Churn") {
			t.Errorf("output missing expected headers:\n%s", out)
		}
		if !strings.Contains(out, "src/") || !strings.Contains(out, "docs/") {
			t.Errorf("output missing expected directories:\n%s", out)
		}
	})

	t.Run("hotspot alias with json format", func(t *testing.T) {
		out, err := runChurnCmdInDir(dir, []string{"hotspot", "-f", "json"})
		if err != nil {
			t.Fatalf("hotspot alias failed: %v\nOutput: %s", err, out)
		}

		var report model.ChurnReport
		if err := json.Unmarshal([]byte(out), &report); err != nil {
			t.Fatalf("failed to parse JSON report: %v\nOutput: %s", err, out)
		}

		// src/ was touched in commit 1, commit 2, commit 3 -> 3 commits
		// docs/ was touched in commit 1 -> 1 commit
		if len(report.Entries) != 2 {
			t.Fatalf("expected 2 entries (src/, docs/), got %d", len(report.Entries))
		}

		// Default sort is commits descending -> src/ must be first
		srcEntry := report.Entries[0]
		if srcEntry.Path != "src/" {
			t.Errorf("expected first entry to be src/, got %s", srcEntry.Path)
		}
		if srcEntry.Commits != 4 {
			t.Errorf("expected src/ commits=4, got %d", srcEntry.Commits)
		}
		if srcEntry.Files != 2 { // main.go and util.go
			t.Errorf("expected src/ unique files=2, got %d", srcEntry.Files)
		}

		docsEntry := report.Entries[1]
		if docsEntry.Path != "docs/" {
			t.Errorf("expected second entry to be docs/, got %s", docsEntry.Path)
		}
		if docsEntry.Commits != 1 {
			t.Errorf("expected docs/ commits=1, got %d", docsEntry.Commits)
		}
	})

	t.Run("markdown format with stat", func(t *testing.T) {
		out, err := runChurnCmdInDir(dir, []string{"churn", "-f", "markdown", "--stat"})
		if err != nil {
			t.Fatalf("churn markdown failed: %v\nOutput: %s", err, out)
		}

		if !strings.Contains(out, "| Directory | Commits | Files | Added | Deleted | Churn | Percent | Graph |") {
			t.Errorf("markdown header mismatch:\n%s", out)
		}
		if !strings.Contains(out, "| `src/` |") {
			t.Errorf("markdown missing src/ row:\n%s", out)
		}
		if !strings.Contains(out, "| **TOTAL** |") {
			t.Errorf("markdown missing TOTAL row:\n%s", out)
		}
	})

	t.Run("fast mode and top flag", func(t *testing.T) {
		out, err := runChurnCmdInDir(dir, []string{"churn", "--fast", "--top", "1", "-f", "json"})
		if err != nil {
			t.Fatalf("churn fast failed: %v\nOutput: %s", err, out)
		}

		var report model.ChurnReport
		if err := json.Unmarshal([]byte(out), &report); err != nil {
			t.Fatalf("failed to parse JSON report: %v\nOutput: %s", err, out)
		}

		if len(report.Entries) != 1 {
			t.Fatalf("expected 1 entry due to --top 1, got %d", len(report.Entries))
		}
		if report.Entries[0].Path != "src/" {
			t.Errorf("expected top entry to be src/, got %s", report.Entries[0].Path)
		}
		// Fast mode: Added and Deleted are 0
		if report.Entries[0].Added != 0 || report.Entries[0].Deleted != 0 {
			t.Errorf("expected 0 added/deleted in fast mode, got added=%d deleted=%d", report.Entries[0].Added, report.Entries[0].Deleted)
		}
		if report.Entries[0].Commits != 4 {
			t.Errorf("expected 4 commits in fast mode, got %d", report.Entries[0].Commits)
		}
	})

	t.Run("max count limit", func(t *testing.T) {
		out, err := runChurnCmdInDir(dir, []string{"churn", "-n", "1", "-f", "json"})
		if err != nil {
			t.Fatalf("churn -n 1 failed: %v\nOutput: %s", err, out)
		}

		var report model.ChurnReport
		if err := json.Unmarshal([]byte(out), &report); err != nil {
			t.Fatalf("failed to parse JSON report: %v\nOutput: %s", err, out)
		}

		if report.Summary.TotalCommits != 1 {
			t.Errorf("expected 1 commit processed with -n 1, got %d", report.Summary.TotalCommits)
		}
	})
}
