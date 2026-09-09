package gitutil

import (
	"os"
	"testing"
	"github.com/sakurahilljp/git-dirstat/pkg/model"
)

func TestResolveCommits_AdditionalSpecs(t *testing.T) {
	dir, repo := createTempGitRepo(t)
	defer os.RemoveAll(dir)

	c1 := addCommit(t, repo, dir, "f1.txt", []byte("1"), "commit 1")
	c2 := addCommit(t, repo, dir, "f2.txt", []byte("2"), "commit 2")

	// TwoDot
	resolved, err := ResolveCommits(repo, model.CommitOptions{
		SpecType:   model.CommitSpecTwoDot,
		FromCommit: c1.String()[:7],
		ToCommit:   c2.String()[:7],
	})
	if err != nil {
		t.Fatalf("unexpected error for TwoDot: %v", err)
	}
	if resolved.FromCommit.Hash != c1 || resolved.ToCommit.Hash != c2 {
		t.Errorf("TwoDot resolved incorrectly")
	}

	// TwoCommits
	resolved, err = ResolveCommits(repo, model.CommitOptions{
		SpecType:   model.CommitSpecTwoCommits,
		FromCommit: c1.String()[:7],
		ToCommit:   c2.String()[:7],
	})
	if err != nil {
		t.Fatalf("unexpected error for TwoCommits: %v", err)
	}
	if resolved.FromCommit.Hash != c1 || resolved.ToCommit.Hash != c2 {
		t.Errorf("TwoCommits resolved incorrectly")
	}

	// Invalid specs
	_, err = ResolveCommits(repo, model.CommitOptions{
		SpecType: model.CommitSpecType(999),
	})
	if err == nil {
		t.Errorf("expected error for unknown spec type")
	}

	// TwoDot invalid from
	_, err = ResolveCommits(repo, model.CommitOptions{
		SpecType:   model.CommitSpecTwoDot,
		FromCommit: "invalid",
		ToCommit:   c2.String()[:7],
	})
	if err == nil {
		t.Errorf("expected error for invalid from commit")
	}
    
    // TwoDot invalid to
	_, err = ResolveCommits(repo, model.CommitOptions{
		SpecType:   model.CommitSpecTwoDot,
		FromCommit: c1.String()[:7],
		ToCommit:   "invalid",
	})
	if err == nil {
		t.Errorf("expected error for invalid to commit")
	}

	// ThreeDot invalid C1
	_, err = ResolveCommits(repo, model.CommitOptions{
		SpecType: model.CommitSpecThreeDot,
		Commit1:  "invalid",
		Commit2:  c2.String()[:7],
	})
	if err == nil {
		t.Errorf("expected error for invalid c1 in threedot")
	}

	// ThreeDot invalid C2
	_, err = ResolveCommits(repo, model.CommitOptions{
		SpecType: model.CommitSpecThreeDot,
		Commit1:  c1.String()[:7],
		Commit2:  "invalid",
	})
	if err == nil {
		t.Errorf("expected error for invalid c2 in threedot")
	}
}

func TestResolveCommits_WorkingTree(t *testing.T) {
	dir, repo := createTempGitRepo(t)
	defer os.RemoveAll(dir)
	addCommit(t, repo, dir, "f1.txt", []byte("1"), "commit 1")

	resolved, err := ResolveCommits(repo, model.CommitOptions{
		SpecType: model.CommitSpecWorkingTree,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resolved.IsWorkingTree {
		t.Errorf("expected IsWorkingTree=true")
	}
}

func TestResolveCommits_EmptyRepo(t *testing.T) {
	dir, repo := createTempGitRepo(t)
	defer os.RemoveAll(dir)

	// HEAD is invalid because there are no commits
	_, err := ResolveCommits(repo, model.CommitOptions{
		SpecType: model.CommitSpecWorkingTree,
	})
	if err == nil {
		t.Errorf("expected error resolving HEAD on empty repo")
	}
}
