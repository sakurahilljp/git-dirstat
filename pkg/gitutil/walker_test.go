package gitutil

import (
	"os"
	"testing"
	"time"

	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/sakurahilljp/git-dirstat/pkg/model"
)

func TestWalkCommitHistoryStream_Basic(t *testing.T) {
	dir, repo := createTempGitRepo(t)
	defer os.RemoveAll(dir)

	h1 := addCommit(t, repo, dir, "a/file1.go", []byte("line1\nline2\n"), "commit 1")
	h2 := addCommit(t, repo, dir, "b/file2.go", []byte("hello\n"), "commit 2")
	h3 := addCommit(t, repo, dir, "a/file1.go", []byte("line1\nline2\nline3\n"), "commit 3")

	var visitedCommits []string
	var allDiffs []model.FileDiff

	err := WalkCommitHistoryStream(repo, ChurnWalkOptions{
		NoMerges: true,
	}, nil, func(c *object.Commit, diffs []model.FileDiff) error {
		visitedCommits = append(visitedCommits, c.Hash.String())
		allDiffs = append(allDiffs, diffs...)
		return nil
	})

	if err != nil {
		t.Fatalf("WalkCommitHistoryStream failed: %v", err)
	}

	if len(visitedCommits) != 3 {
		t.Fatalf("expected 3 commits, got %d", len(visitedCommits))
	}

	// Verify that commit 1, 2, 3 were processed
	if visitedCommits[0] != h3.String() || visitedCommits[1] != h2.String() || visitedCommits[2] != h1.String() {
		t.Errorf("unexpected commit visit order: %v", visitedCommits)
	}

	// Total diffs:
	// c3: a/file1.go (+1)
	// c2: b/file2.go (+1)
	// c1 (root): a/file1.go (+2)
	totalAdded := 0
	for _, d := range allDiffs {
		totalAdded += d.Added
	}
	if totalAdded != 4 {
		t.Errorf("expected 4 total added lines, got %d", totalAdded)
	}
}

func TestWalkCommitHistoryStream_FastMode(t *testing.T) {
	dir, repo := createTempGitRepo(t)
	defer os.RemoveAll(dir)

	_ = addCommit(t, repo, dir, "pkg/util.go", []byte("package util\nline2\n"), "c1")
	_ = addCommit(t, repo, dir, "pkg/util.go", []byte("package util\nline2\nline3\n"), "c2")

	var diffs []model.FileDiff
	err := WalkCommitHistoryStream(repo, ChurnWalkOptions{
		Fast: true,
	}, nil, func(c *object.Commit, ds []model.FileDiff) error {
		diffs = append(diffs, ds...)
		return nil
	})

	if err != nil {
		t.Fatalf("WalkCommitHistoryStream failed: %v", err)
	}

	if len(diffs) != 2 {
		t.Fatalf("expected 2 diff entries, got %d", len(diffs))
	}
	for _, d := range diffs {
		if d.Added != 0 || d.Deleted != 0 {
			t.Errorf("fast mode should not calculate line counts, got added=%d deleted=%d", d.Added, d.Deleted)
		}
	}
}

func TestWalkCommitHistoryStream_MaxCount(t *testing.T) {
	dir, repo := createTempGitRepo(t)
	defer os.RemoveAll(dir)

	_ = addCommit(t, repo, dir, "f1.txt", []byte("1"), "c1")
	_ = addCommit(t, repo, dir, "f2.txt", []byte("2"), "c2")
	_ = addCommit(t, repo, dir, "f3.txt", []byte("3"), "c3")

	count := 0
	err := WalkCommitHistoryStream(repo, ChurnWalkOptions{
		MaxCount: 2,
	}, nil, func(c *object.Commit, ds []model.FileDiff) error {
		count++
		return nil
	})

	if err != nil {
		t.Fatalf("WalkCommitHistoryStream failed: %v", err)
	}

	if count != 2 {
		t.Errorf("expected 2 commits processed with MaxCount=2, got %d", count)
	}
}

func TestWalkCommitHistoryStream_DateFilter(t *testing.T) {
	dir, repo := createTempGitRepo(t)
	defer os.RemoveAll(dir)

	_ = addCommit(t, repo, dir, "f1.txt", []byte("1"), "c1")

	past := time.Now().Add(-1 * time.Hour)
	future := time.Now().Add(1 * time.Hour)

	// Since in future -> should process 0 commits
	count := 0
	err := WalkCommitHistoryStream(repo, ChurnWalkOptions{
		Since: &future,
	}, nil, func(c *object.Commit, ds []model.FileDiff) error {
		count++
		return nil
	})
	if err != nil {
		t.Fatalf("WalkCommitHistoryStream failed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 commits for future Since, got %d", count)
	}

	// Since in past -> should process 1 commit
	count = 0
	err = WalkCommitHistoryStream(repo, ChurnWalkOptions{
		Since: &past,
	}, nil, func(c *object.Commit, ds []model.FileDiff) error {
		count++
		return nil
	})
	if err != nil {
		t.Fatalf("WalkCommitHistoryStream failed: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 commit for past Since, got %d", count)
	}
}

func TestWalkCommitHistoryStream_CommitRange(t *testing.T) {
	dir, repo := createTempGitRepo(t)
	defer os.RemoveAll(dir)

	h1 := addCommit(t, repo, dir, "f1.txt", []byte("1"), "c1")
	h2 := addCommit(t, repo, dir, "f2.txt", []byte("2"), "c2")
	h3 := addCommit(t, repo, dir, "f3.txt", []byte("3"), "c3")

	rangeStr := h1.String() + ".." + h3.String()
	var visited []string

	err := WalkCommitHistoryStream(repo, ChurnWalkOptions{
		CommitRange: rangeStr,
	}, nil, func(c *object.Commit, ds []model.FileDiff) error {
		visited = append(visited, c.Hash.String())
		return nil
	})

	if err != nil {
		t.Fatalf("WalkCommitHistoryStream failed: %v", err)
	}

	// c1..c3 should contain c3 and c2, but NOT c1
	if len(visited) != 2 {
		t.Fatalf("expected 2 commits in range, got %d: %v", len(visited), visited)
	}
	if visited[0] != h3.String() || visited[1] != h2.String() {
		t.Errorf("unexpected commits: %v", visited)
	}
}

