package gitutil

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	fdiff "github.com/go-git/go-git/v5/plumbing/format/diff"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/utils/binary"
	"github.com/go-git/go-git/v5/utils/diff"
	"github.com/sakurahilljp/git-dirstat/pkg/model"
	"github.com/sergi/go-diff/diffmatchpatch"
)

// DiffCommits computes file diffs between two commits.
func DiffCommits(fromCommit, toCommit *object.Commit) ([]model.FileDiff, error) {
	fromTree, err := fromCommit.Tree()
	if err != nil {
		return nil, model.NewRuntimeError("failed to get tree for commit %s: %w", fromCommit.Hash, err)
	}
	toTree, err := toCommit.Tree()
	if err != nil {
		return nil, model.NewRuntimeError("failed to get tree for commit %s: %w", toCommit.Hash, err)
	}

	changes, err := object.DiffTree(fromTree, toTree)
	if err != nil {
		return nil, model.NewRuntimeError("failed to diff trees: %w", err)
	}

	patch, err := changes.Patch()
	if err != nil {
		return nil, model.NewRuntimeError("failed to create patch: %w", err)
	}

	var results []model.FileDiff
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
			// Moved / renamed without rename detection:
			// delete from old path, add to new path
			results = append(results, model.FileDiff{
				Path:     from.Path(),
				Added:    0,
				Deleted:  deleted,
				IsBinary: isBinary,
			})
			results = append(results, model.FileDiff{
				Path:     to.Path(),
				Added:    added,
				Deleted:  0,
				IsBinary: isBinary,
			})
		} else {
			filePath := ""
			if to != nil {
				filePath = to.Path()
			} else if from != nil {
				filePath = from.Path()
			}

			results = append(results, model.FileDiff{
				Path:     filePath,
				Added:    added,
				Deleted:  deleted,
				IsBinary: isBinary,
			})
		}
	}

	return results, nil
}

// DiffWorkingTree computes file diffs between HEAD commit and the working tree.
// Untracked files are excluded. Staged and unstaged changes are aggregated.
func DiffWorkingTree(repo *git.Repository, headCommit *object.Commit, repoRoot string) ([]model.FileDiff, error) {
	wt, err := repo.Worktree()
	if err != nil {
		return nil, model.NewRuntimeError("failed to get worktree: %w", err)
	}

	status, err := wt.Status()
	if err != nil {
		return nil, model.NewRuntimeError("failed to get worktree status: %w", err)
	}

	headTree, err := headCommit.Tree()
	if err != nil {
		return nil, model.NewRuntimeError("failed to get HEAD tree: %w", err)
	}

	var results []model.FileDiff

	for filePath, fileStatus := range status {
		// Untracked files are excluded
		if fileStatus.Staging == git.Untracked && fileStatus.Worktree == git.Untracked {
			continue
		}
		if fileStatus.Staging == '?' || fileStatus.Worktree == '?' {
			continue
		}
		if fileStatus.Staging == git.Unmodified && fileStatus.Worktree == git.Unmodified {
			continue
		}

		// Normalize filePath to forward slash
		normPath := filepath.ToSlash(filePath)
		fullDiskPath := filepath.Join(repoRoot, filepath.FromSlash(normPath))

		// Check if file exists in HEAD
		var headFile *object.File
		var headContent string
		var headIsBinary bool
		headExists := false
		if f, err := headTree.File(normPath); err == nil && f != nil {
			headFile = f
			headExists = true
			isBin, _ := f.IsBinary()
			headIsBinary = isBin
			if !isBin {
				headContent, _ = f.Contents()
			}
		}

		// Check if file exists on disk
		diskExists := false
		var diskContent string
		var diskIsBinary bool
		if fileInfo, statErr := os.Stat(fullDiskPath); statErr == nil && !fileInfo.IsDir() {
			diskExists = true
			bin, binErr := isFileBinary(fullDiskPath)
			if binErr == nil && bin {
				diskIsBinary = true
			} else {
				contentBytes, readErr := os.ReadFile(fullDiskPath)
				if readErr == nil {
					diskContent = string(contentBytes)
				}
			}
		}

		// Case 1: Deleted file (exists in HEAD, missing on disk)
		if headExists && !diskExists {
			if headIsBinary {
				results = append(results, model.FileDiff{
					Path:     normPath,
					Added:    0,
					Deleted:  0,
					IsBinary: true,
				})
			} else {
				results = append(results, model.FileDiff{
					Path:     normPath,
					Added:    0,
					Deleted:  countLines(headContent),
					IsBinary: false,
				})
			}
			continue
		}

		// Case 2: Added file (missing in HEAD, exists on disk and tracked/staged)
		if !headExists && diskExists {
			if diskIsBinary {
				results = append(results, model.FileDiff{
					Path:     normPath,
					Added:    0,
					Deleted:  0,
					IsBinary: true,
				})
			} else {
				results = append(results, model.FileDiff{
					Path:     normPath,
					Added:    countLines(diskContent),
					Deleted:  0,
					IsBinary: false,
				})
			}
			continue
		}

		// Case 3: Modified file (exists in both)
		if headExists && diskExists {
			if headIsBinary || diskIsBinary {
				diskBytes, _ := os.ReadFile(fullDiskPath)
				if headReader, err := headFile.Reader(); err == nil {
					headBytes, _ := io.ReadAll(headReader)
					_ = headReader.Close()
					if !bytes.Equal(headBytes, diskBytes) {
						results = append(results, model.FileDiff{
							Path:     normPath,
							Added:    0,
							Deleted:  0,
							IsBinary: true,
						})
					}
				}
			} else {
				if headContent == diskContent {
					// Content identical, skip
					continue
				}
				diffs := diff.Do(headContent, diskContent)
				var added, deleted int
				for _, d := range diffs {
					switch d.Type {
					case diffmatchpatch.DiffInsert:
						added += countLines(d.Text)
					case diffmatchpatch.DiffDelete:
						deleted += countLines(d.Text)
					}
				}
				results = append(results, model.FileDiff{
					Path:     normPath,
					Added:    added,
					Deleted:  deleted,
					IsBinary: false,
				})
			}
		}
	}

	return results, nil
}

func countLines(s string) int {
	if len(s) == 0 {
		return 0
	}
	lines := strings.Count(s, "\n")
	if s[len(s)-1] != '\n' {
		lines++
	}
	return lines
}

func isFileBinary(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()
	return binary.IsBinary(f)
}
