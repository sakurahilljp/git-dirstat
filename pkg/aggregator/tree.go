package aggregator

import (
	"math"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sakurahilljp/git-dirstat/pkg/filter"
	"github.com/sakurahilljp/git-dirstat/pkg/model"
)

// TreeAggregator accumulates file diffs on-the-fly into a hierarchical tree structure.
type TreeAggregator struct {
	opts          AggregatorOptions
	targetPrefix  string
	pathFilter    *filter.PathFilter
	totalFiles    map[string]struct{}
	summary       model.Summary
	normTarget    string
	displayTarget string
	rawDiffs      []model.FileDiff
}

// NewTreeAggregator creates a new TreeAggregator initialized with the provided options.
func NewTreeAggregator(opts AggregatorOptions) (*TreeAggregator, error) {
	normTarget, displayTarget, err := NormalizeTarget(opts.RepoRoot, opts.Cwd, opts.TargetPath)
	if err != nil {
		return nil, err
	}

	targetPrefix := ""
	if normTarget != "" && normTarget != "." {
		targetPrefix = strings.TrimSuffix(normTarget, "/") + "/"
	}

	pathFilter := filter.NewPathFilter(targetPrefix, opts.Exclude)

	return &TreeAggregator{
		opts:          opts,
		targetPrefix:  targetPrefix,
		pathFilter:    pathFilter,
		totalFiles:    make(map[string]struct{}),
		normTarget:    normTarget,
		displayTarget: displayTarget,
		rawDiffs:      make([]model.FileDiff, 0),
	}, nil
}

// PathFilter returns the PathFilter matching the target and exclude options of this aggregator.
func (t *TreeAggregator) PathFilter() *filter.PathFilter {
	return t.pathFilter
}

// Consume processes an individual file diff and collects it.
func (t *TreeAggregator) Consume(d model.FileDiff) error {
	filePath := filepath.ToSlash(d.Path)

	if !t.pathFilter.ShouldProcess(filePath) {
		return nil
	}

	t.totalFiles[filePath] = struct{}{}
	t.summary.TotalAdded += d.Added
	t.summary.TotalDeleted += d.Deleted
	t.rawDiffs = append(t.rawDiffs, d)

	return nil
}

// internalTreeNode represents an intermediate building node.
type internalTreeNode struct {
	name      string
	relPath   string // target-relative path, e.g. "pkg/model/"
	fullPath  string // repository-relative path
	isDir     bool
	depth     int
	filesSet  map[string]struct{}
	added     int
	deleted   int
	fileDiffs []model.FileDiff
	children  map[string]*internalTreeNode
}

func newInternalNode(name, relPath, fullPath string, isDir bool, depth int) *internalTreeNode {
	return &internalTreeNode{
		name:      name,
		relPath:   relPath,
		fullPath:  fullPath,
		isDir:     isDir,
		depth:     depth,
		filesSet:  make(map[string]struct{}),
		children:  make(map[string]*internalTreeNode),
		fileDiffs: make([]model.FileDiff, 0),
	}
}

// Result builds the full tree, computes rolled-up stats, sorts children, and sets initial expansion state.
func (t *TreeAggregator) Result() (*model.TreeReport, error) {
	t.summary.TotalFiles = len(t.totalFiles)
	t.summary.Net = t.summary.TotalAdded - t.summary.TotalDeleted

	rootName := t.displayTarget
	if rootName == "" {
		rootName = "."
	}
	root := newInternalNode(rootName, "", t.normTarget, true, 0)

	// Build Trie hierarchy
	for _, d := range t.rawDiffs {
		filePath := filepath.ToSlash(d.Path)
		relPath := filePath
		if t.targetPrefix != "" {
			relPath = strings.TrimPrefix(filePath, t.targetPrefix)
		}
		relPath = strings.TrimPrefix(relPath, "/")

		// Split relPath into segments
		cleanRel := path.Clean(relPath)
		segments := strings.Split(cleanRel, "/")
		if len(segments) == 0 || (len(segments) == 1 && segments[0] == ".") {
			continue
		}

		curr := root
		curr.filesSet[filePath] = struct{}{}
		curr.added += d.Added
		curr.deleted += d.Deleted

		accumPath := ""
		for i, seg := range segments {
			if seg == "" || seg == "." {
				continue
			}
			isLeaf := (i == len(segments)-1)
			if accumPath == "" {
				accumPath = seg
			} else {
				accumPath = accumPath + "/" + seg
			}

			fullAccumPath := accumPath
			if t.targetPrefix != "" {
				fullAccumPath = t.targetPrefix + accumPath
			}

			if !isLeaf {
				accumPathWithSlash := accumPath + "/"
				fullAccumPathWithSlash := fullAccumPath + "/"
				child, exists := curr.children[seg]
				if !exists {
					child = newInternalNode(seg+"/", accumPathWithSlash, fullAccumPathWithSlash, true, curr.depth+1)
					curr.children[seg] = child
				}
				child.filesSet[filePath] = struct{}{}
				child.added += d.Added
				child.deleted += d.Deleted
				curr = child
			} else {
				// Leaf is a file
				child, exists := curr.children[seg]
				if !exists {
					child = newInternalNode(seg, accumPath, fullAccumPath, false, curr.depth+1)
					curr.children[seg] = child
				}
				child.filesSet[filePath] = struct{}{}
				child.added += d.Added
				child.deleted += d.Deleted
				// Record this diff under the current directory node's fileDiffs
				curr.fileDiffs = append(curr.fileDiffs, d)
			}
		}
	}

	totalChanges := t.summary.TotalAdded + t.summary.TotalDeleted

	// Convert internal nodes to model.TreeNode recursively and sort
	treeRoot := t.convertAndSort(root, totalChanges)

	return &model.TreeReport{
		Target:  t.displayTarget,
		Summary: t.summary,
		Root:    treeRoot,
	}, nil
}

func (t *TreeAggregator) convertAndSort(in *internalTreeNode, totalChanges int) *model.TreeNode {
	percent := 0.0
	nodeChanges := in.added + in.deleted
	if totalChanges > 0 {
		percent = math.Round((float64(nodeChanges)/float64(totalChanges))*1000) / 10
	}

	out := &model.TreeNode{
		Name:      in.name,
		Path:      in.fullPath,
		IsDir:     in.isDir,
		Files:     len(in.filesSet),
		Added:     in.added,
		Deleted:   in.deleted,
		Net:       in.added - in.deleted,
		Percent:   percent,
		Depth:     in.depth,
		Expanded:  in.depth < t.opts.Depth, // Expand up to Depth option (Option A)
		FileDiffs: in.fileDiffs,
	}

	if in.depth == 0 {
		// Root is always expanded
		out.Expanded = true
	}

	var children []*model.TreeNode
	for _, childIn := range in.children {
		children = append(children, t.convertAndSort(childIn, totalChanges))
	}

	sortTreeNodes(children, t.opts.Sort, t.opts.Reverse)
	out.Children = children

	return out
}

func sortTreeNodes(nodes []*model.TreeNode, sortField string, reverse bool) {
	sort.Slice(nodes, func(i, j int) bool {
		a, b := nodes[i], nodes[j]

		// Directory before file preference or uniform? Standard is directory then file, or purely by metric.
		// In dirstat, dirs usually are primary. If IsDir differs, group dirs first?
		// Keeping directories first makes tree navigation intuitive.
		if a.IsDir != b.IsDir {
			return a.IsDir // dirs first
		}

		var cmp bool
		switch sortField {
		case model.SortPath:
			cmp = a.Name < b.Name

		case model.SortFiles:
			if a.Files != b.Files {
				cmp = a.Files > b.Files
			} else {
				cmp = a.Name < b.Name
			}

		case model.SortDeleted:
			if a.Deleted != b.Deleted {
				cmp = a.Deleted > b.Deleted
			} else {
				cmp = a.Name < b.Name
			}

		case model.SortNet:
			if a.Net != b.Net {
				cmp = a.Net > b.Net
			} else {
				cmp = a.Name < b.Name
			}

		case model.SortPercent:
			aChanges := a.Added + a.Deleted
			bChanges := b.Added + b.Deleted
			if aChanges != bChanges {
				cmp = aChanges > bChanges
			} else {
				cmp = a.Name < b.Name
			}

		case model.SortAdded:
			fallthrough
		default:
			if a.Added != b.Added {
				cmp = a.Added > b.Added
			} else {
				cmp = a.Name < b.Name
			}
		}

		if reverse {
			return !cmp
		}
		return cmp
	})
}
