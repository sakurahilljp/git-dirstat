package aggregator

import (
	"math"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sakurahilljp/git-dirstat/pkg/filter"
	"github.com/sakurahilljp/git-dirstat/pkg/model"
)

// ChurnAggregatorOptions contains options for filtering and aggregating churn history.
type ChurnAggregatorOptions struct {
	RepoRoot    string
	Cwd         string
	TargetPath  string
	Depth       int
	Sort        string
	Reverse     bool
	Top         int
	Exclude     []string
	Since       string
	Until       string
	CommitRange string
}

type churnBucketAccumulator struct {
	path    string
	isRoot  bool
	commits map[string]struct{}
	files   map[string]struct{}
	added   int
	deleted int
}

// ChurnAggregator accumulates churn statistics per directory across commits.
type ChurnAggregator struct {
	opts          ChurnAggregatorOptions
	targetPrefix  string
	pathFilter    *filter.PathFilter
	buckets       map[string]*churnBucketAccumulator
	totalCommits  map[string]struct{}
	totalFiles    map[string]struct{}
	totalAdded    int
	totalDeleted  int
	normTarget    string
	displayTarget string
}

// NewChurnAggregator creates a new ChurnAggregator.
func NewChurnAggregator(opts ChurnAggregatorOptions) (*ChurnAggregator, error) {
	normTarget, displayTarget, err := NormalizeTarget(opts.RepoRoot, opts.Cwd, opts.TargetPath)
	if err != nil {
		return nil, err
	}

	targetPrefix := ""
	if normTarget != "" && normTarget != "." {
		targetPrefix = strings.TrimSuffix(normTarget, "/") + "/"
	}

	pathFilter := filter.NewPathFilter(targetPrefix, opts.Exclude)

	if opts.Sort == "" {
		opts.Sort = model.SortCommits
	}

	return &ChurnAggregator{
		opts:          opts,
		targetPrefix:  targetPrefix,
		pathFilter:    pathFilter,
		buckets:       make(map[string]*churnBucketAccumulator),
		totalCommits:  make(map[string]struct{}),
		totalFiles:    make(map[string]struct{}),
		normTarget:    normTarget,
		displayTarget: displayTarget,
	}, nil
}

// PathFilter returns the PathFilter for pre-filtering files before diff computation.
func (c *ChurnAggregator) PathFilter() *filter.PathFilter {
	return c.pathFilter
}

// ConsumeCommit processes all file diffs belonging to a single commit.
func (c *ChurnAggregator) ConsumeCommit(commitHash string, diffs []model.FileDiff) error {
	commitMatched := false

	for _, d := range diffs {
		filePath := filepath.ToSlash(d.Path)

		if !c.pathFilter.ShouldProcess(filePath) {
			continue
		}

		commitMatched = true

		relPath := filePath
		if c.targetPrefix != "" {
			relPath = strings.TrimPrefix(filePath, c.targetPrefix)
		}

		bucketKey, isRoot := getBucketKey(c.targetPrefix, relPath, c.opts.Depth)

		b, exists := c.buckets[bucketKey]
		if !exists {
			b = &churnBucketAccumulator{
				path:    bucketKey,
				isRoot:  isRoot,
				commits: make(map[string]struct{}),
				files:   make(map[string]struct{}),
			}
			c.buckets[bucketKey] = b
		}

		b.commits[commitHash] = struct{}{}
		b.files[filePath] = struct{}{}
		b.added += d.Added
		b.deleted += d.Deleted

		c.totalFiles[filePath] = struct{}{}
		c.totalAdded += d.Added
		c.totalDeleted += d.Deleted
	}

	if commitMatched {
		c.totalCommits[commitHash] = struct{}{}
	}

	return nil
}

// Result finalizes and produces the aggregated ChurnReport.
func (c *ChurnAggregator) Result() (*model.ChurnReport, error) {
	totalCommitsCount := len(c.totalCommits)
	totalChurn := c.totalAdded + c.totalDeleted

	var entries []model.ChurnEntry
	for _, b := range c.buckets {
		churn := b.added + b.deleted
		commitsCount := len(b.commits)

		var percent float64
		if c.opts.Sort == model.SortCommits {
			if totalCommitsCount > 0 {
				percent = math.Round((float64(commitsCount)/float64(totalCommitsCount)*100)*10) / 10
			}
		} else {
			if totalChurn > 0 {
				percent = math.Round((float64(churn)/float64(totalChurn)*100)*10) / 10
			}
		}

		entries = append(entries, model.ChurnEntry{
			Path:    b.path,
			IsRoot:  b.isRoot,
			Commits: commitsCount,
			Files:   len(b.files),
			Added:   b.added,
			Deleted: b.deleted,
			Churn:   churn,
			Percent: percent,
		})
	}

	sortChurnEntries(entries, c.opts.Sort)

	if c.opts.Reverse {
		for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
			entries[i], entries[j] = entries[j], entries[i]
		}
	}

	if c.opts.Top > 0 && len(entries) > c.opts.Top {
		entries = entries[:c.opts.Top]
	}

	summary := model.ChurnSummary{
		TotalCommits: totalCommitsCount,
		TotalFiles:   len(c.totalFiles),
		TotalAdded:   c.totalAdded,
		TotalDeleted: c.totalDeleted,
		TotalChurn:   totalChurn,
	}

	return &model.ChurnReport{
		Target:      c.displayTarget,
		Depth:       c.opts.Depth,
		Since:       c.opts.Since,
		Until:       c.opts.Until,
		CommitRange: c.opts.CommitRange,
		Summary:     summary,
		Entries:     entries,
	}, nil
}

func sortChurnEntries(entries []model.ChurnEntry, sortField string) {
	sort.Slice(entries, func(i, j int) bool {
		a, b := entries[i], entries[j]

		switch sortField {
		case model.SortPath:
			return a.Path < b.Path

		case model.SortCommits:
			if a.Commits != b.Commits {
				return a.Commits > b.Commits
			}
			if a.Churn != b.Churn {
				return a.Churn > b.Churn
			}
			return a.Path < b.Path

		case model.SortChurn:
			if a.Churn != b.Churn {
				return a.Churn > b.Churn
			}
			if a.Commits != b.Commits {
				return a.Commits > b.Commits
			}
			return a.Path < b.Path

		case model.SortFiles:
			if a.Files != b.Files {
				return a.Files > b.Files
			}
			return a.Path < b.Path

		case model.SortAdded:
			if a.Added != b.Added {
				return a.Added > b.Added
			}
			return a.Path < b.Path

		case model.SortDeleted:
			if a.Deleted != b.Deleted {
				return a.Deleted > b.Deleted
			}
			return a.Path < b.Path

		case model.SortPercent:
			if a.Percent != b.Percent {
				return a.Percent > b.Percent
			}
			return a.Path < b.Path

		default:
			// Default to SortCommits
			if a.Commits != b.Commits {
				return a.Commits > b.Commits
			}
			return a.Path < b.Path
		}
	})
}
