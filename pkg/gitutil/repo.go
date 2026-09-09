package gitutil

import (
	"path/filepath"

	"github.com/go-git/go-git/v5"
	"github.com/sakurahilljp/git-dirstat/pkg/model"
)

// RepositoryContext wraps the opened repository and its root directory.
type RepositoryContext struct {
	Repo     *git.Repository
	RepoRoot string
}

// OpenRepository traverses upward from startPath to find and open the git repository.
func OpenRepository(startPath string) (*RepositoryContext, error) {
	repo, err := git.PlainOpenWithOptions(startPath, &git.PlainOpenOptions{
		DetectDotGit: true,
	})
	if err != nil {
		return nil, model.NewRuntimeError("failed to open git repository: %w", err)
	}

	wt, err := repo.Worktree()
	var repoRoot string
	if err == nil && wt != nil && wt.Filesystem != nil {
		repoRoot = filepath.Clean(wt.Filesystem.Root())
	} else {
		// Bare repository or direct storage
		repoRoot = filepath.Clean(startPath)
	}

	// Verify that the repository has at least one commit
	_, err = repo.Head()
	if err != nil {
		return nil, model.NewRuntimeError("repository has no commits yet (empty repository): %w", err)
	}

	return &RepositoryContext{
		Repo:     repo,
		RepoRoot: repoRoot,
	}, nil
}
