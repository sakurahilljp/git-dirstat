package gitutil

import (
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/sakurahilljp/git-dirstat/pkg/model"
)

// ResolvedCommits contains the resolved commits to compare.
type ResolvedCommits struct {
	IsWorkingTree bool
	HeadCommit    *object.Commit
	FromCommit    *object.Commit
	ToCommit      *object.Commit
}

// ResolveCommits resolves the commit specifications into git commit objects.
func ResolveCommits(repo *git.Repository, opts model.CommitOptions) (*ResolvedCommits, error) {
	// Always resolve HEAD first to ensure valid reference
	headCommit, err := resolveCommit(repo, "HEAD")
	if err != nil {
		return nil, err
	}

	switch opts.SpecType {
	case model.CommitSpecWorkingTree:
		return &ResolvedCommits{
			IsWorkingTree: true,
			HeadCommit:    headCommit,
			FromCommit:    headCommit,
		}, nil

	case model.CommitSpecCommitToHead:
		from, err := resolveCommit(repo, opts.FromCommit)
		if err != nil {
			return nil, err
		}
		return &ResolvedCommits{
			IsWorkingTree: false,
			HeadCommit:    headCommit,
			FromCommit:    from,
			ToCommit:      headCommit,
		}, nil

	case model.CommitSpecTwoDot, model.CommitSpecTwoCommits:
		from, err := resolveCommit(repo, opts.FromCommit)
		if err != nil {
			return nil, err
		}
		to, err := resolveCommit(repo, opts.ToCommit)
		if err != nil {
			return nil, err
		}
		return &ResolvedCommits{
			IsWorkingTree: false,
			HeadCommit:    headCommit,
			FromCommit:    from,
			ToCommit:      to,
		}, nil

	case model.CommitSpecThreeDot:
		c1, err := resolveCommit(repo, opts.Commit1)
		if err != nil {
			return nil, err
		}
		c2, err := resolveCommit(repo, opts.Commit2)
		if err != nil {
			return nil, err
		}

		bases, err := c1.MergeBase(c2)
		if err != nil || len(bases) == 0 {
			return nil, model.NewRuntimeError("no common ancestor found between %q and %q", opts.Commit1, opts.Commit2)
		}

		return &ResolvedCommits{
			IsWorkingTree: false,
			HeadCommit:    headCommit,
			FromCommit:    bases[0],
			ToCommit:      c2,
		}, nil

	default:
		return nil, model.NewRuntimeError("unknown commit spec type: %v", opts.SpecType)
	}
}

func resolveCommit(repo *git.Repository, refStr string) (*object.Commit, error) {
	hash, err := repo.ResolveRevision(plumbing.Revision(refStr))
	if err != nil {
		return nil, model.NewRuntimeError("failed to resolve revision %q: %w", refStr, err)
	}
	commit, err := repo.CommitObject(*hash)
	if err != nil {
		return nil, model.NewRuntimeError("failed to find commit %q (%s): %w", refStr, hash.String(), err)
	}
	return commit, nil
}
