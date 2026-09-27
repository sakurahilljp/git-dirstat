package gitutil

import (
	"container/list"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	fdiff "github.com/go-git/go-git/v5/plumbing/format/diff"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/sakurahilljp/git-dirstat/pkg/filter"
	"github.com/sakurahilljp/git-dirstat/pkg/model"
)

// ChurnCommitConsumer receives the commit and its file diffs.
type ChurnCommitConsumer func(commit *object.Commit, diffs []model.FileDiff) error

// ChurnWalkOptions controls commit history walking and filtering.
type ChurnWalkOptions struct {
	CommitRange string
	Since       *time.Time
	Until       *time.Time
	MaxCount    int
	NoMerges    bool
	FirstParent bool
	Fast        bool
}

// WalkCommitHistoryStream walks commit history and streams diffs per commit.
func WalkCommitHistoryStream(
	repo *git.Repository,
	opts ChurnWalkOptions,
	pathFilter *filter.PathFilter,
	consumer ChurnCommitConsumer,
) error {
	startCommit, stopCommits, err := resolveCommitRange(repo, opts.CommitRange)
	if err != nil {
		return err
	}

	// Build set of excluded commit hashes if a range like c1..c2 was specified
	stopSet := make(map[string]struct{})
	for _, sc := range stopCommits {
		if err := collectAncestors(sc, stopSet); err != nil {
			return err
		}
	}

	queue := list.New()
	queue.PushBack(startCommit)
	visited := make(map[string]struct{})
	visited[startCommit.Hash.String()] = struct{}{}

	processedCount := 0

	for queue.Len() > 0 {
		elem := queue.Front()
		queue.Remove(elem)
		commit := elem.Value.(*object.Commit)

		commitHashStr := commit.Hash.String()
		if _, isStopped := stopSet[commitHashStr]; isStopped {
			continue
		}

		// Enqueue parents
		numParents := commit.NumParents()
		if opts.FirstParent {
			if numParents > 0 {
				parent, err := commit.Parent(0)
				if err == nil {
					pHash := parent.Hash.String()
					if _, seen := visited[pHash]; !seen {
						visited[pHash] = struct{}{}
						queue.PushBack(parent)
					}
				}
			}
		} else {
			for i := 0; i < numParents; i++ {
				parent, err := commit.Parent(i)
				if err == nil {
					pHash := parent.Hash.String()
					if _, seen := visited[pHash]; !seen {
						visited[pHash] = struct{}{}
						queue.PushBack(parent)
					}
				}
			}
		}

		// Date filtering
		commitTime := commit.Author.When
		if opts.Until != nil && commitTime.After(*opts.Until) {
			continue
		}
		if opts.Since != nil && commitTime.Before(*opts.Since) {
			continue
		}

		// Merge commit handling
		if opts.NoMerges && numParents > 1 {
			continue
		}

		// Process commit diff
		diffs, err := extractCommitDiffs(commit, pathFilter, opts.Fast)
		if err != nil {
			return err
		}

		if len(diffs) > 0 {
			if err := consumer(commit, diffs); err != nil {
				return err
			}
		}

		processedCount++
		if opts.MaxCount > 0 && processedCount >= opts.MaxCount {
			break
		}
	}

	return nil
}

func resolveCommitRange(repo *git.Repository, commitRange string) (*object.Commit, []*object.Commit, error) {
	commitRange = strings.TrimSpace(commitRange)
	if commitRange == "" {
		head, err := resolveCommit(repo, "HEAD")
		if err != nil {
			return nil, nil, err
		}
		return head, nil, nil
	}

	if strings.Contains(commitRange, "..") {
		parts := strings.Split(commitRange, "..")
		if len(parts) == 2 {
			fromRef := strings.TrimSpace(parts[0])
			toRef := strings.TrimSpace(parts[1])
			if toRef == "" {
				toRef = "HEAD"
			}
			toCommit, err := resolveCommit(repo, toRef)
			if err != nil {
				return nil, nil, err
			}
			var stopCommits []*object.Commit
			if fromRef != "" {
				fromCommit, err := resolveCommit(repo, fromRef)
				if err != nil {
					return nil, nil, err
				}
				stopCommits = append(stopCommits, fromCommit)
			}
			return toCommit, stopCommits, nil
		}
	}

	// Single ref
	c, err := resolveCommit(repo, commitRange)
	if err != nil {
		return nil, nil, err
	}
	return c, nil, nil
}

func collectAncestors(commit *object.Commit, set map[string]struct{}) error {
	queue := list.New()
	queue.PushBack(commit)
	set[commit.Hash.String()] = struct{}{}

	for queue.Len() > 0 {
		elem := queue.Front()
		queue.Remove(elem)
		c := elem.Value.(*object.Commit)

		for i := 0; i < c.NumParents(); i++ {
			p, err := c.Parent(i)
			if err != nil {
				continue
			}
			h := p.Hash.String()
			if _, ok := set[h]; !ok {
				set[h] = struct{}{}
				queue.PushBack(p)
			}
		}
	}
	return nil
}

func extractCommitDiffs(commit *object.Commit, pathFilter *filter.PathFilter, fast bool) ([]model.FileDiff, error) {
	toTree, err := commit.Tree()
	if err != nil {
		return nil, model.NewRuntimeError("failed to get tree for commit %s: %w", commit.Hash, err)
	}

	if commit.NumParents() == 0 {
		// Root / initial commit: all files are newly added
		return extractRootCommitDiffs(toTree, pathFilter, fast)
	}

	parent, err := commit.Parent(0)
	if err != nil {
		return nil, model.NewRuntimeError("failed to get parent for commit %s: %w", commit.Hash, err)
	}
	fromTree, err := parent.Tree()
	if err != nil {
		return nil, model.NewRuntimeError("failed to get tree for parent commit %s: %w", parent.Hash, err)
	}

	changes, err := object.DiffTree(fromTree, toTree)
	if err != nil {
		return nil, model.NewRuntimeError("failed to diff trees: %w", err)
	}

	var results []model.FileDiff
	for _, change := range changes {
		fromPath := ""
		if change.From.Name != "" {
			fromPath = filepath.ToSlash(change.From.Name)
		}
		toPath := ""
		if change.To.Name != "" {
			toPath = filepath.ToSlash(change.To.Name)
		}

		if pathFilter != nil && !pathFilter.ShouldProcessChange(fromPath, toPath) {
			continue
		}

		filePath := toPath
		if filePath == "" {
			filePath = fromPath
		}

		if fast {
			// Fast mode skips Myers line diff computation
			results = append(results, model.FileDiff{
				Path: filePath,
			})
			continue
		}

		patch, err := change.Patch()
		if err != nil {
			return nil, model.NewRuntimeError("failed to create patch for %s: %w", filePath, err)
		}

		for _, fp := range patch.FilePatches() {
			from, to := fp.Files()
			isBinary := fp.IsBinary()
			var added, deleted int
			if !isBinary {
				for _, chunk := range fp.Chunks() {
					switch chunk.Type() {
					case fdiff.Add:
						added += countLines(chunk.Content())
					case fdiff.Delete:
						deleted += countLines(chunk.Content())
					}
				}
			}

			if from != nil && to != nil && from.Path() != to.Path() {
				if pathFilter == nil || pathFilter.ShouldProcess(from.Path()) {
					results = append(results, model.FileDiff{
						Path:     from.Path(),
						Deleted:  deleted,
						IsBinary: isBinary,
					})
				}
				if pathFilter == nil || pathFilter.ShouldProcess(to.Path()) {
					results = append(results, model.FileDiff{
						Path:     to.Path(),
						Added:    added,
						IsBinary: isBinary,
					})
				}
			} else {
				target := filePath
				if to != nil {
					target = to.Path()
				} else if from != nil {
					target = from.Path()
				}
				if pathFilter == nil || pathFilter.ShouldProcess(target) {
					results = append(results, model.FileDiff{
						Path:     target,
						Added:    added,
						Deleted:  deleted,
						IsBinary: isBinary,
					})
				}
			}
		}
	}

	return results, nil
}

func extractRootCommitDiffs(tree *object.Tree, pathFilter *filter.PathFilter, fast bool) ([]model.FileDiff, error) {
	var results []model.FileDiff
	fileIter := tree.Files()
	defer fileIter.Close()

	err := fileIter.ForEach(func(f *object.File) error {
		path := filepath.ToSlash(f.Name)
		if pathFilter != nil && !pathFilter.ShouldProcess(path) {
			return nil
		}

		if fast {
			results = append(results, model.FileDiff{Path: path})
			return nil
		}

		isBin, err := f.IsBinary()
		if err != nil {
			isBin = false
		}

		added := 0
		if !isBin {
			r, err := f.Reader()
			if err == nil {
				defer r.Close()
				added, _ = countLinesFromReader(r)
			}
		}

		results = append(results, model.FileDiff{
			Path:     path,
			Added:    added,
			Deleted:  0,
			IsBinary: isBin,
		})
		return nil
	})

	return results, err
}


