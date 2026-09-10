package aggregator

import (
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sakurahilljp/git-dirstat/pkg/filter"
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

// StreamAggregator accumulates file diffs on-the-fly without buffering all diffs in a slice.
type StreamAggregator struct {
	opts          AggregatorOptions
	targetPrefix  string
	pathFilter    *filter.PathFilter
	buckets       map[string]*bucketAccumulator
	totalFiles    map[string]struct{}
	summary       model.Summary
	normTarget    string
	displayTarget string
}

// NewStreamAggregator creates a new StreamAggregator initialized with the provided options.
func NewStreamAggregator(opts AggregatorOptions) (*StreamAggregator, error) {
	normTarget, displayTarget, err := NormalizeTarget(opts.RepoRoot, opts.Cwd, opts.TargetPath)
	if err != nil {
		return nil, err
	}

	targetPrefix := ""
	if normTarget != "" && normTarget != "." {
		targetPrefix = strings.TrimSuffix(normTarget, "/") + "/"
	}

	pathFilter := filter.NewPathFilter(targetPrefix, opts.Exclude)

	return &StreamAggregator{
		opts:          opts,
		targetPrefix:  targetPrefix,
		pathFilter:    pathFilter,
		buckets:       make(map[string]*bucketAccumulator),
		totalFiles:    make(map[string]struct{}),
		normTarget:    normTarget,
		displayTarget: displayTarget,
	}, nil
}

// PathFilter returns the PathFilter matching the target and exclude options of this aggregator.
func (s *StreamAggregator) PathFilter() *filter.PathFilter {
	return s.pathFilter
}

// Consume processes an individual file diff and aggregates it into the appropriate bucket.
func (s *StreamAggregator) Consume(d model.FileDiff) error {
	filePath := filepath.ToSlash(d.Path)

	// Filter out if outside target directory or matching exclude patterns
	if !s.pathFilter.ShouldProcess(filePath) {
		return nil
	}

	// Calculate relative path from target
	relPath := filePath
	if s.targetPrefix != "" {
		relPath = strings.TrimPrefix(filePath, s.targetPrefix)
	}

	// Determine bucket key
	bucketKey, isRoot := getBucketKey(s.targetPrefix, relPath, s.opts.Depth)

	b, exists := s.buckets[bucketKey]
	if !exists {
		b = &bucketAccumulator{
			path:    bucketKey,
			isRoot:  isRoot,
			files:   make(map[string]struct{}),
			added:   0,
			deleted: 0,
		}
		s.buckets[bucketKey] = b
	}

	b.files[filePath] = struct{}{}
	b.added += d.Added
	b.deleted += d.Deleted

	s.totalFiles[filePath] = struct{}{}
	s.summary.TotalAdded += d.Added
	s.summary.TotalDeleted += d.Deleted

	return nil
}

// Result finalizes the aggregation, sorts entries, and produces the final model.Report.
func (s *StreamAggregator) Result() (*model.Report, error) {
	var entries []model.Entry

	for _, b := range s.buckets {
		entry := model.Entry{
			Path:    b.path,
			IsRoot:  b.isRoot,
			Files:   len(b.files),
			Added:   b.added,
			Deleted: b.deleted,
			Net:     b.added - b.deleted,
		}
		entries = append(entries, entry)
	}

	s.summary.TotalFiles = len(s.totalFiles)
	s.summary.Net = s.summary.TotalAdded - s.summary.TotalDeleted

	if entries == nil {
		entries = []model.Entry{}
	}

	// Sort entries
	sortEntries(entries, s.opts.Sort)

	// Reverse if requested
	if s.opts.Reverse {
		for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
			entries[i], entries[j] = entries[j], entries[i]
		}
	}

	return &model.Report{
		Target:  s.displayTarget,
		Depth:   s.opts.Depth,
		Summary: s.summary,
		Entries: entries,
	}, nil
}

// Aggregate takes raw file diffs and produces the aggregated model.Report.
// Retained for backward compatibility.
func Aggregate(diffs []model.FileDiff, opts AggregatorOptions) (*model.Report, error) {
	agg, err := NewStreamAggregator(opts)
	if err != nil {
		return nil, err
	}
	for _, d := range diffs {
		if err := agg.Consume(d); err != nil {
			return nil, err
		}
	}
	return agg.Result()
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
