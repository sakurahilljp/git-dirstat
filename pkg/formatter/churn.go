package formatter

import (
	"io"
	"strings"

	"github.com/sakurahilljp/git-dirstat/pkg/model"
)

// ChurnFormatter formats a model.ChurnReport to an io.Writer.
type ChurnFormatter interface {
	FormatChurn(w io.Writer, report *model.ChurnReport) error
}

// FormatChurnEntryPath returns the formatted path for churn display.
func FormatChurnEntryPath(entry model.ChurnEntry, target string) string {
	if entry.IsRoot {
		if target == "." || target == "" {
			return "(root files)"
		}
		cleanTarget := strings.TrimSuffix(target, "/")
		return cleanTarget + "/ (root files)"
	}
	return entry.Path
}
