package model

const (
	SortFiles   = "files"
	SortAdded   = "added"
	SortDeleted = "deleted"
	SortNet     = "net"
	SortPath    = "path"
	SortPercent = "percent"
)

const (
	FormatTable    = "table"
	FormatJSON     = "json"
	FormatCSV      = "csv"
	FormatTSV      = "tsv"
	FormatMarkdown = "markdown"
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
