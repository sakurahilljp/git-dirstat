package formatter

import (
	"io"
	"strings"

	"github.com/sakurahilljp/git-dirstat/pkg/model"
)

// Formatter formats a model.Report to an io.Writer.
type Formatter interface {
	Format(w io.Writer, report *model.Report) error
}

// FormatEntryPath returns the formatted path for table/csv/tsv display.
func FormatEntryPath(entry model.Entry, target string) string {
	if entry.IsRoot {
		if target == "." || target == "" {
			return "(root files)"
		}
		cleanTarget := strings.TrimSuffix(target, "/")
		return cleanTarget + "/ (root files)"
	}
	return entry.Path
}
