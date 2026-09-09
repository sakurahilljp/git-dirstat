package aggregator

import (
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/sakurahilljp/git-dirstat/pkg/model"
)

// AggregatorOptions contains options for filtering and aggregating diffs.
type AggregatorOptions struct {
	RepoRoot   string
	Cwd        string
	TargetPath string
	Depth      int
	Sort       string
	Reverse    bool
	Exclude    []string
}

type bucketAccumulator struct {
	path    string
	isRoot  bool
	files   map[string]struct{}
	added   int
	deleted int
}

// Aggregate takes raw file diffs and produces the aggregated model.Report.
func Aggregate(diffs []model.FileDiff, opts AggregatorOptions) (*model.Report, error) {
	// 1. Normalize target path relative to repository root
	normTarget, displayTarget, err := NormalizeTarget(opts.RepoRoot, opts.Cwd, opts.TargetPath)
	if err != nil {
		return nil, err
	}

	targetPrefix := ""
	if normTarget != "" && normTarget != "." {
		targetPrefix = strings.TrimSuffix(normTarget, "/") + "/"
	}

	// 2. Filter diffs and group into buckets
	buckets := make(map[string]*bucketAccumulator)

	for _, d := range diffs {
		filePath := filepath.ToSlash(d.Path)

		// Filter out by exclude patterns
		if isExcluded(filePath, opts.Exclude) {
			continue
		}

		// Filter out if outside target directory
		if targetPrefix != "" {
			if !strings.HasPrefix(filePath, targetPrefix) {
				continue
			}
		}

		// Calculate relative path from target
		relPath := filePath
		if targetPrefix != "" {
			relPath = strings.TrimPrefix(filePath, targetPrefix)
		}

		// Determine bucket key
		bucketKey, isRoot := getBucketKey(targetPrefix, relPath, opts.Depth)

		b, exists := buckets[bucketKey]
		if !exists {
			b = &bucketAccumulator{
				path:    bucketKey,
				isRoot:  isRoot,
				files:   make(map[string]struct{}),
				added:   0,
				deleted: 0,
			}
			buckets[bucketKey] = b
		}

		b.files[filePath] = struct{}{}
		b.added += d.Added
		b.deleted += d.Deleted
	}

	// 3. Convert buckets to entries and calculate summary
	var entries []model.Entry
	summary := model.Summary{}

	// To count total unique files across the entire target scope
	totalFiles := make(map[string]struct{})

	for _, b := range buckets {
		entry := model.Entry{
			Path:    b.path,
			IsRoot:  b.isRoot,
			Files:   len(b.files),
			Added:   b.added,
			Deleted: b.deleted,
			Net:     b.added - b.deleted,
		}
		entries = append(entries, entry)

		for f := range b.files {
			totalFiles[f] = struct{}{}
		}
		summary.TotalAdded += b.added
		summary.TotalDeleted += b.deleted
	}

	summary.TotalFiles = len(totalFiles)
	summary.Net = summary.TotalAdded - summary.TotalDeleted

	// If no entries, ensure entries is empty non-nil slice
	if entries == nil {
		entries = []model.Entry{}
	}

	// 4. Sort entries
	sortEntries(entries, opts.Sort)

	// 5. Reverse if requested
	if opts.Reverse {
		for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
			entries[i], entries[j] = entries[j], entries[i]
		}
	}

	return &model.Report{
		Target:  displayTarget,
		Depth:   opts.Depth,
		Summary: summary,
		Entries: entries,
	}, nil
}

// NormalizeTarget computes the repository-root-relative target path and its display string.
func NormalizeTarget(repoRoot, cwd, target string) (normTarget, displayTarget string, err error) {
	var absTarget string
	if filepath.IsAbs(target) {
		absTarget = filepath.Clean(target)
	} else {
		absTarget = filepath.Clean(filepath.Join(cwd, target))
	}

	relToRepo, err := filepath.Rel(repoRoot, absTarget)
	if err != nil {
		return "", "", model.NewInputError("failed to resolve target path relative to repository: %w", err)
	}

	cleanRel := filepath.ToSlash(filepath.Clean(relToRepo))
	if cleanRel == "." || cleanRel == "" {
		return ".", ".", nil
	}

	if strings.HasPrefix(cleanRel, "..") {
		return "", "", model.NewInputError("target path %q is outside repository root %q", target, repoRoot)
	}

	// Format with trailing slash for directories
	dirWithSlash := strings.TrimSuffix(cleanRel, "/") + "/"
	return dirWithSlash, dirWithSlash, nil
}

func getBucketKey(targetPrefix, relPath string, depth int) (bucketKey string, isRoot bool) {
	relDir, _ := path.Split(relPath)
	if relDir == "" {
		// Root file directly under target
		if targetPrefix == "" {
			return ".", true
		}
		return targetPrefix, true
	}

	// Subdirectory file
	relDir = strings.TrimSuffix(relDir, "/")
	segments := strings.Split(relDir, "/")

	depthCount := depth
	if len(segments) < depthCount {
		depthCount = len(segments)
	}

	slicedRelDir := strings.Join(segments[:depthCount], "/") + "/"
	return targetPrefix + slicedRelDir, false
}

func isExcluded(filePath string, excludePatterns []string) bool {
	for _, pat := range excludePatterns {
		pat = filepath.ToSlash(pat)
		if matched, err := doublestar.Match(pat, filePath); err == nil && matched {
			return true
		}
		// If pattern contains no slash, check matching on file basename
		if !strings.Contains(pat, "/") {
			if matched, err := doublestar.Match(pat, path.Base(filePath)); err == nil && matched {
				return true
			}
		}
	}
	return false
}

func sortEntries(entries []model.Entry, sortField string) {
	sort.Slice(entries, func(i, j int) bool {
		a, b := entries[i], entries[j]

		switch sortField {
		case model.SortPath:
			return a.Path < b.Path

		case model.SortFiles:
			if a.Files != b.Files {
				return a.Files > b.Files
			}
			return a.Path < b.Path

		case model.SortDeleted:
			if a.Deleted != b.Deleted {
				return a.Deleted > b.Deleted
			}
			return a.Path < b.Path

		case model.SortNet:
			if a.Net != b.Net {
				return a.Net > b.Net
			}
			return a.Path < b.Path

		case model.SortAdded:
			fallthrough
		default:
			if a.Added != b.Added {
				return a.Added > b.Added
			}
			return a.Path < b.Path
		}
	})
}
