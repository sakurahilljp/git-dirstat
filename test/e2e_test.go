package test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/sakurahilljp/git-dirstat/cmd"
	"github.com/sakurahilljp/git-dirstat/pkg/model"
)

func setupTestRepo(t *testing.T) (string, *git.Repository) {
	t.Helper()
	dir, err := os.MkdirTemp("", "git-dirstat-e2e-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	repo, err := git.PlainInit(dir, false)
	if err != nil {
		t.Fatalf("failed to init repo: %v", err)
	}

	return dir, repo
}

func commitFile(t *testing.T, repo *git.Repository, dir, relPath, content string, msg string) plumbing.Hash {
	t.Helper()
	fullPath := filepath.Join(dir, relPath)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		t.Fatalf("failed to mkdir: %v", err)
	}
	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	wt, err := repo.Worktree()
	if err != nil {
		t.Fatalf("failed to get worktree: %v", err)
	}
	if _, err := wt.Add(relPath); err != nil {
		t.Fatalf("failed to add file: %v", err)
	}

	hash, err := wt.Commit(msg, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test User",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		t.Fatalf("failed to commit: %v", err)
	}
	return hash
}

func runCmdInDir(dir string, args ...string) (string, error) {
	origDir, _ := os.Getwd()
	_ = os.Chdir(dir)
	defer func() { _ = os.Chdir(origDir) }()

	root := cmd.NewRootCommand()
	var outBuf bytes.Buffer
	root.SetOut(&outBuf)
	root.SetArgs(args)

	err := root.Execute()
	return outBuf.String(), err
}

func TestE2E_Comprehensive(t *testing.T) {
	dir, repo := setupTestRepo(t)
	defer os.RemoveAll(dir)

	// Initial commit (c1) on main branch
	c1 := commitFile(t, repo, dir, "src/components/button.tsx", "line 1\nline 2\nline 3\n", "initial button")
	commitFile(t, repo, dir, "src/index.ts", "import button\n", "initial index")
	commitFile(t, repo, dir, "README.md", "# Project\n", "initial readme")
	_ = c1

	// Create branch "feature" from c1
	wt, _ := repo.Worktree()
	featureBranch := plumbing.NewBranchReferenceName("feature")
	headRef, _ := repo.Head()
	_ = repo.Storer.SetReference(plumbing.NewReferenceFromStrings(featureBranch.String(), headRef.Hash().String()))

	// On main: commit c2
	c2 := commitFile(t, repo, dir, "src/services/api.ts", "line 1\nline 2\n", "add service on main")

	// Checkout feature branch and commit c3
	_ = wt.Checkout(&git.CheckoutOptions{
		Branch: featureBranch,
	})
	c3 := commitFile(t, repo, dir, "src/components/modal.tsx", "modal 1\nmodal 2\nmodal 3\n", "add modal on feature")
	_ = c3

	// 1. Three-dot range (main...feature): common ancestor is c1. Diff c1 vs feature (c3).
	// Expect modal.tsx to be present in diff, but api.ts should NOT be present.
	t.Run("three-dot range", func(t *testing.T) {
		out, err := runCmdInDir(dir, c2.String()[:7]+"...feature", "-t", "src/", "-f", "json")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		var rep model.Report
		if err := json.Unmarshal([]byte(out), &rep); err != nil {
			t.Fatalf("failed to parse json: %v\nOutput: %s", err, out)
		}
		foundModal := false
		for _, e := range rep.Entries {
			if e.Path == "src/components/" {
				foundModal = true
			}
			if e.Path == "src/services/" {
				t.Errorf("src/services/ should not be present in feature PR diff")
			}
		}
		if !foundModal {
			t.Errorf("src/components/ not found in three-dot report: %+v", rep)
		}
	})

	// 2. Working tree diff (0 args)
	t.Run("working tree diff staged and unstaged", func(t *testing.T) {
		// Modify button.tsx without staging (unstaged change)
		btnPath := filepath.Join(dir, "src/components/button.tsx")
		_ = os.WriteFile(btnPath, []byte("line 1\nline 2 modified\nline 3\nline 4\n"), 0644)

		// Create a new file and stage it (staged change)
		stagedFile := filepath.Join(dir, "src/staged.ts")
		_ = os.WriteFile(stagedFile, []byte("export const a = 1;\n"), 0644)
		_, _ = wt.Add("src/staged.ts")

		// Create an untracked file (should be ignored)
		untrackedFile := filepath.Join(dir, "src/untracked.ts")
		_ = os.WriteFile(untrackedFile, []byte("untracked\n"), 0644)

		out, err := runCmdInDir(dir, "-t", "src/", "-f", "json")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var rep model.Report
		if err := json.Unmarshal([]byte(out), &rep); err != nil {
			t.Fatalf("failed to parse json: %v\nOutput: %s", err, out)
		}

		// Untracked file must not affect root files count or files
		// button.tsx is in src/components/
		// staged.ts is root file in src/
		hasComponents := false
		hasRoot := false
		for _, e := range rep.Entries {
			if e.Path == "src/components/" {
				hasComponents = true
			}
			if e.IsRoot {
				hasRoot = true
				if e.Files != 1 { // only staged.ts, untracked ignored
					t.Errorf("expected 1 root file, got %d", e.Files)
				}
			}
		}
		if !hasComponents || !hasRoot {
			t.Errorf("expected components and root files in report: %+v", rep)
		}
	})

	// 3. Exclude pattern
	t.Run("exclude pattern", func(t *testing.T) {
		out, err := runCmdInDir(dir, "-t", "src/", "-e", "src/components/**", "-f", "json")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		var rep model.Report
		_ = json.Unmarshal([]byte(out), &rep)
		for _, e := range rep.Entries {
			if e.Path == "src/components/" {
				t.Errorf("src/components/ should have been excluded")
			}
		}
	})

	// 4. CSV & TSV Formats
	t.Run("csv format", func(t *testing.T) {
		out, err := runCmdInDir(dir, "-t", "src/", "-f", "csv")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.HasPrefix(out, "path,files,added,deleted,net\n") {
			t.Errorf("unexpected csv header: %s", out)
		}
	})

	// 5. User Input Error (Exit Code 2)
	t.Run("exit code 2 on invalid depth", func(t *testing.T) {
		_, err := runCmdInDir(dir, "--depth", "-1")
		if err == nil {
			t.Fatalf("expected error")
		}
		var exitErr *model.ExitCodeError
		if ok := (err != nil); ok {
			if ee, isEE := err.(*model.ExitCodeError); isEE {
				exitErr = ee
			}
		}
		if exitErr == nil || exitErr.Code != 2 {
			t.Errorf("expected exit code 2, got %v", err)
		}
	})

	// 6. Git / Runtime Error (Exit Code 1)
	t.Run("exit code 1 on missing commit", func(t *testing.T) {
		_, err := runCmdInDir(dir, "deadbeef1234567")
		if err == nil {
			t.Fatalf("expected error")
		}
		var exitErr *model.ExitCodeError
		if ee, isEE := err.(*model.ExitCodeError); isEE {
			exitErr = ee
		}
		if exitErr == nil || exitErr.Code != 1 {
			t.Errorf("expected exit code 1, got %v", err)
		}
	})
}
