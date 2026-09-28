package model

// TreeNode represents a node in the directory tree hierarchy.
type TreeNode struct {
	Name      string      `json:"name"`
	Path      string      `json:"path"`
	IsDir     bool        `json:"is_dir"`
	Files     int         `json:"files"`
	Added     int         `json:"added"`
	Deleted   int         `json:"deleted"`
	Net       int         `json:"net"`
	Percent   float64     `json:"percent"`
	Children  []*TreeNode `json:"children,omitempty"`
	Depth     int         `json:"depth"`
	Expanded  bool        `json:"expanded,omitempty"`
	FileDiffs []FileDiff  `json:"file_diffs,omitempty"`
}

// TreeReport holds the full tree report data.
type TreeReport struct {
	Target  string    `json:"target"`
	Summary Summary   `json:"summary"`
	Root    *TreeNode `json:"root"`
}
