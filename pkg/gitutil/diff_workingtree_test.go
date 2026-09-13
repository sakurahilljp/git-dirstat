package gitutil

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sakurahilljp/git-dirstat/pkg/filter"
	"github.com/sakurahilljp/git-dirstat/pkg/model"
)

func TestDiffWorkingTreeStream_AllStatus(t *testing.T) {
	dir, repo := createTempGitRepo(t)
	defer os.RemoveAll(dir)

	addCommit(t, repo, dir, "unmodified.txt", []byte("line1\nline2\n"), "c1")
	addCommit(t, repo, dir, "modified.txt", []byte("line1\nline2\n"), "c2")
	addCommit(t, repo, dir, "modified_staged.txt", []byte("line1\nline2\n"), "c3")
	h4 := addCommit(t, repo, dir, "deleted.txt", []byte("line1\nline2\n"), "c4")

	head, err := repo.CommitObject(h4)
	if err != nil {
		t.Fatalf("failed to get head: %v", err)
	}

	wt, err := repo.Worktree()
	if err != nil {
		t.Fatalf("failed to get wt: %v", err)
	}

	// modify unstaged
	os.WriteFile(filepath.Join(dir, "modified.txt"), []byte("line1\nline2\nline3\n"), 0644)

	// modify staged
	os.WriteFile(filepath.Join(dir, "modified_staged.txt"), []byte("line1\nline2\nline3\n"), 0644)
	wt.Add("modified_staged.txt")

	// delete
	os.Remove(filepath.Join(dir, "deleted.txt"))

	// untracked (should be ignored by our logic)
	os.WriteFile(filepath.Join(dir, "untracked.txt"), []byte("line1\n"), 0644)

	// added to index
	os.WriteFile(filepath.Join(dir, "added.txt"), []byte("line1\nline2\n"), 0644)
	wt.Add("added.txt")

	pf := filter.NewPathFilter("", nil)

	var diffs []model.FileDiff
	err = DiffWorkingTreeStream(repo, head, dir, pf, func(d model.FileDiff) error {
		diffs = append(diffs, d)
		return nil
	})
	if err != nil {
		t.Fatalf("DiffWorkingTreeStream failed: %v", err)
	}

	foundModified := false
	foundModifiedStaged := false
	foundAdded := false
	foundDeleted := false

	for _, d := range diffs {
		if d.Path == "modified.txt" {
			foundModified = true
			if d.Added != 1 || d.Deleted != 0 {
				t.Errorf("modified.txt: want +1 -0, got +%d -%d", d.Added, d.Deleted)
			}
		}
		if d.Path == "modified_staged.txt" {
			foundModifiedStaged = true
			if d.Added != 1 || d.Deleted != 0 {
				t.Errorf("modified_staged.txt: want +1 -0, got +%d -%d", d.Added, d.Deleted)
			}
		}
		if d.Path == "added.txt" {
			foundAdded = true
			if d.Added != 2 || d.Deleted != 0 {
				t.Errorf("added.txt: want +2 -0, got +%d -%d", d.Added, d.Deleted)
			}
		}
		if d.Path == "deleted.txt" {
			foundDeleted = true
			if d.Added != 0 || d.Deleted != 2 {
				t.Errorf("deleted.txt: want +0 -2, got +%d -%d", d.Added, d.Deleted)
			}
		}
	}

	if !foundModified {
		t.Errorf("modified.txt not found in diffs")
	}
	if !foundModifiedStaged {
		t.Errorf("modified_staged.txt not found in diffs")
	}
	if !foundAdded {
		t.Errorf("added.txt not found in diffs")
	}
	if !foundDeleted {
		t.Errorf("deleted.txt not found in diffs")
	}
}
