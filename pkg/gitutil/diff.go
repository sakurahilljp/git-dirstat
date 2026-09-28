package gitutil

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"golang.org/x/sync/errgroup"

	"github.com/go-git/go-git/v5"
	fdiff "github.com/go-git/go-git/v5/plumbing/format/diff"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/utils/binary"
	"github.com/go-git/go-git/v5/utils/diff"
	"github.com/sakurahilljp/git-dirstat/pkg/filter"
	"github.com/sakurahilljp/git-dirstat/pkg/model"
	"github.com/sergi/go-diff/diffmatchpatch"
)

// DiffOptions configures diff computation behavior.
type DiffOptions struct {
	PathFilter *filter.PathFilter
	Workers    int
}

// DiffConsumer is a callback that receives individual file diffs as they are computed.
type DiffConsumer func(diff model.FileDiff) error

// DiffCommits computes file diffs between two commits and returns them as a slice.
// Retained for backward compatibility; internally delegates to DiffCommitsStream.
func DiffCommits(fromCommit, toCommit *object.Commit) ([]model.FileDiff, error) {
	var results []model.FileDiff
	err := DiffCommitsStream(fromCommit, toCommit, DiffOptions{}, func(d model.FileDiff) error {
		results = append(results, d)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return results, nil
}

// DiffCommitsStream streams file diffs between two commits through the consumer callback.
// If opts.PathFilter is non-nil, files outside target or matching exclude patterns are skipped
// before Myers diff patch generation, drastically saving memory and CPU time.
func DiffCommitsStream(
	fromCommit, toCommit *object.Commit,
	opts DiffOptions,
	consumer DiffConsumer,
) error {
	fromTree, err := fromCommit.Tree()
	if err != nil {
		return model.NewRuntimeError("failed to get tree for commit %s: %w", fromCommit.Hash, err)
	}
	toTree, err := toCommit.Tree()
	if err != nil {
		return model.NewRuntimeError("failed to get tree for commit %s: %w", toCommit.Hash, err)
	}

	changes, err := object.DiffTree(fromTree, toTree)
	if err != nil {
		return model.NewRuntimeError("failed to diff trees: %w", err)
	}

	workers := opts.Workers
	if workers <= 0 {
		workers = runtime.GOMAXPROCS(0)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	workerG, ctx := errgroup.WithContext(ctx)
	workerG.SetLimit(workers)

	results := make(chan model.FileDiff, workers*2)

	consumerErr := make(chan error, 1)
	go func() {
		for d := range results {
			if err := consumer(d); err != nil {
				consumerErr <- err
				cancel()
				return
			}
		}
		consumerErr <- nil
	}()

	for _, change := range changes {
		fromPath := ""
		if change.From.Name != "" {
			fromPath = filepath.ToSlash(change.From.Name)
		}
		toPath := ""
		if change.To.Name != "" {
			toPath = filepath.ToSlash(change.To.Name)
		}

		// Pre-filtering
		if opts.PathFilter != nil && !opts.PathFilter.ShouldProcessChange(fromPath, toPath) {
			continue
		}

		fromPathCapture := fromPath
		toPathCapture := toPath

		workerG.Go(func() error {
			patchName := toPathCapture
			if patchName == "" {
				patchName = fromPathCapture
			}
			patch, err := change.Patch()
			if err != nil {
				return model.NewRuntimeError("failed to create patch for %s: %w", patchName, err)
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
					// Moved / renamed without rename detection
					if opts.PathFilter == nil || opts.PathFilter.ShouldProcess(from.Path()) {
						select {
						case <-ctx.Done():
							return ctx.Err()
						case results <- model.FileDiff{Path: from.Path(), Added: 0, Deleted: deleted, IsBinary: isBinary}:
						}
					}
					if opts.PathFilter == nil || opts.PathFilter.ShouldProcess(to.Path()) {
						select {
						case <-ctx.Done():
							return ctx.Err()
						case results <- model.FileDiff{Path: to.Path(), Added: added, Deleted: 0, IsBinary: isBinary}:
						}
					}
				} else {
					filePath := ""
					if to != nil {
						filePath = to.Path()
					} else if from != nil {
						filePath = from.Path()
					}
					if opts.PathFilter == nil || opts.PathFilter.ShouldProcess(filePath) {
						select {
						case <-ctx.Done():
							return ctx.Err()
						case results <- model.FileDiff{Path: filePath, Added: added, Deleted: deleted, IsBinary: isBinary}:
						}
					}
				}
			}
			return nil
		})
	}

	workerErr := workerG.Wait()
	close(results)

	if err := <-consumerErr; err != nil {
		return err
	}
	if workerErr != nil && workerErr != context.Canceled {
		return workerErr
	}

	return nil
}

// DiffWorkingTree computes file diffs between HEAD commit and the working tree.
// Untracked files are excluded. Staged and unstaged changes are aggregated.
// Retained for backward compatibility; internally delegates to DiffWorkingTreeStream.
func DiffWorkingTree(repo *git.Repository, headCommit *object.Commit, repoRoot string) ([]model.FileDiff, error) {
	var results []model.FileDiff
	err := DiffWorkingTreeStream(repo, headCommit, repoRoot, DiffOptions{}, func(d model.FileDiff) error {
		results = append(results, d)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return results, nil
}

// DiffWorkingTreeStream computes working tree file diffs and streams them to consumer.
// Pre-filters files by pathFilter before loading disk or HEAD content.
func DiffWorkingTreeStream(
	repo *git.Repository,
	headCommit *object.Commit,
	repoRoot string,
	opts DiffOptions,
	consumer DiffConsumer,
) error {
	wt, err := repo.Worktree()
	if err != nil {
		return model.NewRuntimeError("failed to get worktree: %w", err)
	}

	status, err := wt.Status()
	if err != nil {
		return model.NewRuntimeError("failed to get worktree status: %w", err)
	}

	headTree, err := headCommit.Tree()
	if err != nil {
		return model.NewRuntimeError("failed to get HEAD tree: %w", err)
	}

	workers := opts.Workers
	if workers <= 0 {
		workers = runtime.GOMAXPROCS(0)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	workerG, ctx := errgroup.WithContext(ctx)
	workerG.SetLimit(workers)

	results := make(chan model.FileDiff, workers*2)

	consumerErr := make(chan error, 1)
	go func() {
		for d := range results {
			if err := consumer(d); err != nil {
				consumerErr <- err
				cancel()
				return
			}
		}
		consumerErr <- nil
	}()

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

		// Early pre-filtering
		if opts.PathFilter != nil && !opts.PathFilter.ShouldProcess(normPath) {
			continue
		}

		normPathCapture := normPath

		workerG.Go(func() error {
			fullDiskPath := filepath.Join(repoRoot, filepath.FromSlash(normPathCapture))

			// Check if file exists in HEAD
			var headFile *object.File
			headExists := false
			headIsBinary := false
			if f, err := headTree.File(normPathCapture); err == nil && f != nil {
				headFile = f
				headExists = true
				isBin, _ := f.IsBinary()
				headIsBinary = isBin
			}

			// Check if file exists on disk
			diskExists := false
			diskIsBinary := false
			var fileInfo os.FileInfo
			if fi, statErr := os.Stat(fullDiskPath); statErr == nil && !fi.IsDir() {
				diskExists = true
				fileInfo = fi
				bin, binErr := isFileBinary(fullDiskPath)
				if binErr == nil && bin {
					diskIsBinary = true
				}
			}

			// Case 1: Deleted file
			if headExists && !diskExists {
				if headIsBinary {
					select {
					case <-ctx.Done():
						return ctx.Err()
					case results <- model.FileDiff{Path: normPathCapture, Added: 0, Deleted: 0, IsBinary: true}:
					}
				} else {
					deletedLines, err := countFileLinesFromObject(headFile)
					if err != nil {
						return model.NewRuntimeError("failed to read deleted file %s: %w", normPathCapture, err)
					}
					select {
					case <-ctx.Done():
						return ctx.Err()
					case results <- model.FileDiff{Path: normPathCapture, Added: 0, Deleted: deletedLines, IsBinary: false}:
					}
				}
				return nil
			}

			// Case 2: Added file
			if !headExists && diskExists {
				if diskIsBinary {
					select {
					case <-ctx.Done():
						return ctx.Err()
					case results <- model.FileDiff{Path: normPathCapture, Added: 0, Deleted: 0, IsBinary: true}:
					}
				} else {
					addedLines, err := countFileLinesFromDisk(fullDiskPath)
					if err != nil {
						return model.NewRuntimeError("failed to read added file %s: %w", normPathCapture, err)
					}
					select {
					case <-ctx.Done():
						return ctx.Err()
					case results <- model.FileDiff{Path: normPathCapture, Added: addedLines, Deleted: 0, IsBinary: false}:
					}
				}
				return nil
			}

			// Case 3: Modified file
			if headExists && diskExists {
				if headIsBinary || diskIsBinary {
					identical, err := compareBinaryFiles(headFile, fullDiskPath, fileInfo)
					if err != nil {
						return model.NewRuntimeError("failed to compare binary files %s: %w", normPathCapture, err)
					}
					if !identical {
						select {
						case <-ctx.Done():
							return ctx.Err()
						case results <- model.FileDiff{Path: normPathCapture, Added: 0, Deleted: 0, IsBinary: true}:
						}
					}
				} else {
					headContent, err := headFile.Contents()
					if err != nil {
						return model.NewRuntimeError("failed to read HEAD content for %s: %w", normPathCapture, err)
					}
					contentBytes, err := os.ReadFile(fullDiskPath)
					if err != nil {
						return model.NewRuntimeError("failed to read disk content for %s: %w", normPathCapture, err)
					}
					diskContent := string(contentBytes)

					if headContent != diskContent {
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
						select {
						case <-ctx.Done():
							return ctx.Err()
						case results <- model.FileDiff{Path: normPathCapture, Added: added, Deleted: deleted, IsBinary: false}:
						}
					}
				}
			}
			return nil
		})
	}

	workerErr := workerG.Wait()
	close(results)

	if err := <-consumerErr; err != nil {
		return err
	}
	if workerErr != nil && workerErr != context.Canceled {
		return workerErr
	}

	return nil
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

func countLinesFromReader(r io.Reader) (int, error) {
	buf := make([]byte, 32*1024)
	count := 0
	hasBytes := false
	lastByte := byte(0)

	for {
		n, err := r.Read(buf)
		if n > 0 {
			hasBytes = true
			count += bytes.Count(buf[:n], []byte{'\n'})
			lastByte = buf[n-1]
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return 0, err
		}
	}

	if hasBytes && lastByte != '\n' {
		count++
	}
	return count, nil
}

func countFileLinesFromObject(f *object.File) (int, error) {
	r, err := f.Reader()
	if err != nil {
		return 0, err
	}
	defer r.Close()
	return countLinesFromReader(r)
}

func countFileLinesFromDisk(path string) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	return countLinesFromReader(f)
}

func compareBinaryFiles(headFile *object.File, diskPath string, fi os.FileInfo) (bool, error) {
	// Fast-path size check: if disk size differs from Git object size, files are different.
	// Note: Pure Go go-git does not execute external Git LFS / smudge filters, so files managed
	// by LFS pointers in Git object DB may differ in size from the actual checked-out binary on disk.
	if fi != nil && headFile.Size != fi.Size() {
		return false, nil
	}

	headReader, err := headFile.Reader()
	if err != nil {
		return false, err
	}
	defer headReader.Close()

	diskFile, err := os.Open(diskPath)
	if err != nil {
		return false, err
	}
	defer diskFile.Close()

	b1 := make([]byte, 8192)
	b2 := make([]byte, 8192)

	for {
		n1, err1 := io.ReadFull(headReader, b1)
		n2, err2 := io.ReadFull(diskFile, b2)

		if n1 != n2 || !bytes.Equal(b1[:n1], b2[:n2]) {
			return false, nil
		}

		if err1 == io.EOF || err1 == io.ErrUnexpectedEOF {
			if err2 == io.EOF || err2 == io.ErrUnexpectedEOF {
				return true, nil
			}
			return false, nil
		}
		if err1 != nil {
			return false, err1
		}
		if err2 != nil {
			return false, err2
		}
	}
}

func isFileBinary(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()
	return binary.IsBinary(f)
}
