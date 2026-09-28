package model

const (
	SortFiles   = "files"
	SortAdded   = "added"
	SortDeleted = "deleted"
	SortNet     = "net"
	SortPath    = "path"
	SortPercent = "percent"
	SortCommits = "commits"
	SortChurn   = "churn"
)

const (
	FormatTable    = "table"
	FormatJSON     = "json"
	FormatCSV      = "csv"
	FormatTSV      = "tsv"
	FormatMarkdown = "markdown"
	FormatTree     = "tree"
)

// Entry represents an aggregated directory or root files bucket.
type Entry struct {
	Path    string  `json:"path"`
	IsRoot  bool    `json:"is_root"`
	Files   int     `json:"files"`
	Added   int     `json:"added"`
	Deleted int     `json:"deleted"`
	Net     int     `json:"net"`
	Percent float64 `json:"percent"`
}

// Summary represents the overall total changes.
type Summary struct {
	TotalFiles   int `json:"total_files"`
	TotalAdded   int `json:"total_added"`
	TotalDeleted int `json:"total_deleted"`
	Net          int `json:"net"`
}

// Report holds the full report data to be rendered by formatters.
type Report struct {
	Target  string  `json:"target"`
	Depth   int     `json:"depth"`
	Summary Summary `json:"summary"`
	Entries []Entry `json:"entries"`
}

// ChurnEntry represents aggregated hotspot metrics for a directory.
type ChurnEntry struct {
	Path    string  `json:"path"`
	IsRoot  bool    `json:"is_root"`
	Commits int     `json:"commits"`
	Files   int     `json:"files"`
	Added   int     `json:"added"`
	Deleted int     `json:"deleted"`
	Churn   int     `json:"churn"`
	Percent float64 `json:"percent"`
}

// ChurnSummary represents overall summary metrics across the history.
type ChurnSummary struct {
	TotalCommits int `json:"total_commits"`
	TotalFiles   int `json:"total_files"`
	TotalAdded   int `json:"total_added"`
	TotalDeleted int `json:"total_deleted"`
	TotalChurn   int `json:"total_churn"`
}

// ChurnReport holds the full churn analysis report data to be rendered by formatters.
type ChurnReport struct {
	Target      string       `json:"target"`
	Depth       int          `json:"depth"`
	Since       string       `json:"since,omitempty"`
	Until       string       `json:"until,omitempty"`
	CommitRange string       `json:"commit_range,omitempty"`
	Summary     ChurnSummary `json:"summary"`
	Entries     []ChurnEntry `json:"entries"`
}

