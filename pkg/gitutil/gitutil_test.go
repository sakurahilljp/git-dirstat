package gitutil

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/sakurahilljp/git-dirstat/pkg/model"
)

func createTempGitRepo(t *testing.T) (string, *git.Repository) {
	t.Helper()
	dir, err := os.MkdirTemp("", "gitutil-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	repo, err := git.PlainInit(dir, false)
	if err != nil {
		t.Fatalf("failed to init git repo: %v", err)
	}

	return dir, repo
}

func addCommit(t *testing.T, repo *git.Repository, dir, relPath string, content []byte, msg string) plumbing.Hash {
	t.Helper()
	fullPath := filepath.Join(dir, relPath)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		t.Fatalf("failed to mkdir: %v", err)
	}
	if err := os.WriteFile(fullPath, content, 0644); err != nil {
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
			Name:  "Test",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		t.Fatalf("failed to commit: %v", err)
	}
	return hash
}

func TestOpenRepository(t *testing.T) {
	// 1. Not a git repository
	tempDir, err := os.MkdirTemp("", "non-git-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	_, err = OpenRepository(tempDir)
	if err == nil {
		t.Errorf("expected error for non-git repository")
	}

	// 2. Empty git repository (no commits)
	repoDir, _ := createTempGitRepo(t)
	defer os.RemoveAll(repoDir)

	_, err = OpenRepository(repoDir)
	if err == nil {
		t.Errorf("expected error for empty repository without commits")
	}
	if exitErr, ok := err.(*model.ExitCodeError); !ok || exitErr.Code != 1 {
		t.Errorf("expected ExitCode 1, got %v", err)
	}

	// 3. Valid repository with a commit
	repo, _ := git.PlainOpen(repoDir)
	addCommit(t, repo, repoDir, "hello.txt", []byte("hello world\n"), "initial commit")

	ctx, err := OpenRepository(repoDir)
	if err != nil {
		t.Fatalf("failed to open valid repository: %v", err)
	}
	realRepoDir, _ := filepath.EvalSymlinks(repoDir)
	realCtxRoot, _ := filepath.EvalSymlinks(ctx.RepoRoot)
	if realCtxRoot != realRepoDir {
		t.Errorf("expected repo root %q, got %q", realRepoDir, realCtxRoot)
	}
}

func TestResolveCommits(t *testing.T) {
	dir, repo := createTempGitRepo(t)
	defer os.RemoveAll(dir)

	c1 := addCommit(t, repo, dir, "file1.txt", []byte("line 1\nline 2\n"), "commit 1")
	c2 := addCommit(t, repo, dir, "file1.txt", []byte("line 1\nline 2\nline 3\n"), "commit 2")

	// Create a tag
	headRef, _ := repo.Head()
	_, _ = repo.CreateTag("v0.1.0", headRef.Hash(), nil)

	// 1. Working tree
	resolved, err := ResolveCommits(repo, model.CommitOptions{SpecType: model.CommitSpecWorkingTree})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resolved.IsWorkingTree || resolved.HeadCommit.Hash != c2 {
		t.Errorf("expected working tree vs c2")
	}

	// 2. CommitToHead (1 arg)
	resolved, err = ResolveCommits(repo, model.CommitOptions{
		SpecType:   model.CommitSpecCommitToHead,
		FromCommit: c1.String()[:7],
		ToCommit:   "HEAD",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved.IsWorkingTree || resolved.FromCommit.Hash != c1 || resolved.ToCommit.Hash != c2 {
		t.Errorf("expected c1 vs c2")
	}

	// 3. Tag resolution
	resolved, err = ResolveCommits(repo, model.CommitOptions{
		SpecType:   model.CommitSpecCommitToHead,
		FromCommit: "v0.1.0",
		ToCommit:   "HEAD",
	})
	if err != nil {
		t.Fatalf("failed to resolve tag: %v", err)
	}
	if resolved.FromCommit.Hash != c2 {
		t.Errorf("expected tag to resolve to c2")
	}

	// 4. Invalid ref -> ExitCode 1
	_, err = ResolveCommits(repo, model.CommitOptions{
		SpecType:   model.CommitSpecCommitToHead,
		FromCommit: "nonexistent_ref_xyz",
	})
	if err == nil {
		t.Fatalf("expected error for nonexistent ref")
	}
	if exitErr, ok := err.(*model.ExitCodeError); !ok || exitErr.Code != 1 {
		t.Errorf("expected ExitCode 1, got %v", err)
	}

	// 5. Relative revision (HEAD~1)
	resolved, err = ResolveCommits(repo, model.CommitOptions{
		SpecType:   model.CommitSpecCommitToHead,
		FromCommit: "HEAD~1",
		ToCommit:   "HEAD",
	})
	if err != nil {
		t.Fatalf("failed to resolve relative revision HEAD~1: %v", err)
	}
	if resolved.FromCommit.Hash != c1 {
		t.Errorf("expected HEAD~1 to resolve to c1")
	}
}

func TestDiffCommits_BinaryAndText(t *testing.T) {
	dir, repo := createTempGitRepo(t)
	defer os.RemoveAll(dir)

	// Commit 1: Text file + binary file
	binaryData1 := []byte{0x00, 0x01, 0x02, 0x03}
	addCommit(t, repo, dir, "data.bin", binaryData1, "initial binary")
	c1 := addCommit(t, repo, dir, "text.txt", []byte("line 1\nline 2\n"), "initial text")

	// Commit 2: Modify text file, modify binary file
	binaryData2 := []byte{0x00, 0x01, 0xFF, 0xFE}
	addCommit(t, repo, dir, "data.bin", binaryData2, "update binary")
	c2 := addCommit(t, repo, dir, "text.txt", []byte("line 1\nline 2 modified\nline 3\n"), "update text")

	commit1, _ := repo.CommitObject(c1)
	commit2, _ := repo.CommitObject(c2)

	diffs, err := DiffCommits(commit1, commit2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	foundBin := false
	foundText := false

	for _, d := range diffs {
		if d.Path == "data.bin" {
			foundBin = true
			if !d.IsBinary || d.Added != 0 || d.Deleted != 0 {
				t.Errorf("binary file diff must have IsBinary:true and Added:0, Deleted:0: %+v", d)
			}
		}
		if d.Path == "text.txt" {
			foundText = true
			if d.IsBinary || d.Added != 2 || d.Deleted != 1 {
				t.Errorf("text file diff unexpected: %+v", d)
			}
		}
	}

	if !foundBin || !foundText {
		t.Errorf("missing expected diff entries: bin=%v, text=%v", foundBin, foundText)
	}
}

func TestDiffWorkingTree_UntrackedAndModified(t *testing.T) {
	dir, repo := createTempGitRepo(t)
	defer os.RemoveAll(dir)

	c1 := addCommit(t, repo, dir, "tracked.txt", []byte("line 1\nline 2\n"), "initial")
	commit1, _ := repo.CommitObject(c1)

	// Modify tracked.txt on disk (unstaged)
	_ = os.WriteFile(filepath.Join(dir, "tracked.txt"), []byte("line 1\nline 2\nline 3\n"), 0644)

	// Add new untracked file (should be excluded)
	_ = os.WriteFile(filepath.Join(dir, "untracked.txt"), []byte("should be ignored\n"), 0644)

	// Add new staged file
	_ = os.WriteFile(filepath.Join(dir, "staged.txt"), []byte("staged content\n"), 0644)
	wt, _ := repo.Worktree()
	_, _ = wt.Add("staged.txt")

	diffs, err := DiffWorkingTree(repo, commit1, dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	foundTracked := false
	foundStaged := false

	for _, d := range diffs {
		if d.Path == "untracked.txt" {
			t.Errorf("untracked.txt must not be present in working tree diff")
		}
		if d.Path == "tracked.txt" {
			foundTracked = true
			if d.Added != 1 || d.Deleted != 0 {
				t.Errorf("unexpected tracked.txt diff: %+v", d)
			}
		}
		if d.Path == "staged.txt" {
			foundStaged = true
			if d.Added != 1 || d.Deleted != 0 {
				t.Errorf("unexpected staged.txt diff: %+v", d)
			}
		}
	}

	if !foundTracked || !foundStaged {
		t.Errorf("expected tracked and staged diffs, got: %+v", diffs)
	}
}

func TestResolveCommits_NoCommonAncestor(t *testing.T) {
	dir, repo := createTempGitRepo(t)
	defer os.RemoveAll(dir)

	c1 := addCommit(t, repo, dir, "branch1.txt", []byte("b1\n"), "commit 1")

	// Create a true orphan commit directly via storer without parent
	emptyTree := &object.Tree{}
	treeObj := repo.Storer.NewEncodedObject()
	_ = emptyTree.Encode(treeObj)
	treeHash, _ := repo.Storer.SetEncodedObject(treeObj)

	commit := &object.Commit{
		Author:       object.Signature{Name: "Test", Email: "test@example.com", When: time.Now()},
		Committer:    object.Signature{Name: "Test", Email: "test@example.com", When: time.Now()},
		Message:      "orphan commit",
		TreeHash:     treeHash,
		ParentHashes: []plumbing.Hash{},
	}
	commitObj := repo.Storer.NewEncodedObject()
	_ = commit.Encode(commitObj)
	c2, _ := repo.Storer.SetEncodedObject(commitObj)

	_, err := ResolveCommits(repo, model.CommitOptions{
		SpecType: model.CommitSpecThreeDot,
		Commit1:  c1.String()[:7],
		Commit2:  c2.String()[:7],
	})
	if err == nil {
		t.Fatalf("expected error for commits with no common ancestor")
	}
	if exitErr, ok := err.(*model.ExitCodeError); !ok || exitErr.Code != 1 {
		t.Errorf("expected ExitCode 1, got %v", err)
	}
}
