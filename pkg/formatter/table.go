package formatter

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/mattn/go-isatty"
	"github.com/sakurahilljp/git-dirstat/pkg/model"
)

const (
	colorGreen = "\x1b[32m"
	colorRed   = "\x1b[31m"
	colorReset = "\x1b[0m"
)

type TableFormatter struct {
	NoColor bool
}

func NewTableFormatter(noColor bool) *TableFormatter {
	return &TableFormatter{
		NoColor: noColor,
	}
}

func (f *TableFormatter) shouldUseColor(w io.Writer) bool {
	if f.NoColor {
		return false
	}
	// If w is os.Stdout, check whether it is a TTY
	if file, ok := w.(*os.File); ok {
		fd := file.Fd()
		return isatty.IsTerminal(fd) || isatty.IsCygwinTerminal(fd)
	}
	return false
}

func (f *TableFormatter) Format(w io.Writer, report *model.Report) error {
	useColor := f.shouldUseColor(w)

	// Determine column widths
	dirWidth := 28
	filesWidth := 10
	addedWidth := 11
	deletedWidth := 11
	netWidth := 12

	type rowData struct {
		dir     string
		files   string
		added   string
		deleted string
		net     string
		netVal  int
	}

	var rows []rowData
	for _, e := range report.Entries {
		dirStr := FormatEntryPath(e, report.Target)
		filesStr := strconv.Itoa(e.Files)
		addedStr := strconv.Itoa(e.Added)
		deletedStr := strconv.Itoa(e.Deleted)
		netStr := formatNet(e.Net)

		if len(dirStr) > dirWidth {
			dirWidth = len(dirStr)
		}
		if len(filesStr) > filesWidth {
			filesWidth = len(filesStr)
		}
		if len(addedStr) > addedWidth {
			addedWidth = len(addedStr)
		}
		if len(deletedStr) > deletedWidth {
			deletedWidth = len(deletedStr)
		}
		if len(netStr) > netWidth {
			netWidth = len(netStr)
		}

		rows = append(rows, rowData{
			dir:     dirStr,
			files:   filesStr,
			added:   addedStr,
			deleted: deletedStr,
			net:     netStr,
			netVal:  e.Net,
		})
	}

	totalFilesStr := strconv.Itoa(report.Summary.TotalFiles)
	totalAddedStr := strconv.Itoa(report.Summary.TotalAdded)
	totalDeletedStr := strconv.Itoa(report.Summary.TotalDeleted)
	totalNetStr := formatNet(report.Summary.Net)

	if len(totalFilesStr) > filesWidth {
		filesWidth = len(totalFilesStr)
	}
	if len(totalAddedStr) > addedWidth {
		addedWidth = len(totalAddedStr)
	}
	if len(totalDeletedStr) > deletedWidth {
		deletedWidth = len(totalDeletedStr)
	}
	if len(totalNetStr) > netWidth {
		netWidth = len(totalNetStr)
	}

	totalWidth := dirWidth + 1 + filesWidth + 1 + addedWidth + 1 + deletedWidth + 1 + netWidth
	sep := strings.Repeat("-", totalWidth)

	// Target line
	fmt.Fprintf(w, "Target: %s (Depth: %d)\n\n", report.Target, report.Depth)

	// Header line
	fmt.Fprintf(w, "%-*s %*s %*s %*s %*s\n",
		dirWidth, "Directory",
		filesWidth, "Files",
		addedWidth, "Added",
		deletedWidth, "Deleted",
		netWidth, "Net",
	)
	fmt.Fprintln(w, sep)

	// Data rows
	for _, r := range rows {
		addedColored := r.added
		deletedColored := r.deleted
		netColored := r.net

		if useColor {
			if r.added != "0" {
				addedColored = colorGreen + r.added + colorReset
			}
			if r.deleted != "0" {
				deletedColored = colorRed + r.deleted + colorReset
			}
			if r.netVal > 0 {
				netColored = colorGreen + r.net + colorReset
			} else if r.netVal < 0 {
				netColored = colorRed + r.net + colorReset
			}
		}

		fmt.Fprintf(w, "%-*s %*s %s%s %s%s %s%s\n",
			dirWidth, r.dir,
			filesWidth, r.files,
			strings.Repeat(" ", addedWidth-len(r.added)), addedColored,
			strings.Repeat(" ", deletedWidth-len(r.deleted)), deletedColored,
			strings.Repeat(" ", netWidth-len(r.net)), netColored,
		)
	}

	if len(rows) > 0 {
		fmt.Fprintln(w, sep)
	}

	// Total row
	totalAddedColored := totalAddedStr
	totalDeletedColored := totalDeletedStr
	totalNetColored := totalNetStr
	if useColor {
		if report.Summary.TotalAdded != 0 {
			totalAddedColored = colorGreen + totalAddedStr + colorReset
		}
		if report.Summary.TotalDeleted != 0 {
			totalDeletedColored = colorRed + totalDeletedStr + colorReset
		}
		if report.Summary.Net > 0 {
			totalNetColored = colorGreen + totalNetStr + colorReset
		} else if report.Summary.Net < 0 {
			totalNetColored = colorRed + totalNetStr + colorReset
		}
	}

	fmt.Fprintf(w, "%-*s %*s %s%s %s%s %s%s\n",
		dirWidth, "TOTAL",
		filesWidth, totalFilesStr,
		strings.Repeat(" ", addedWidth-len(totalAddedStr)), totalAddedColored,
		strings.Repeat(" ", deletedWidth-len(totalDeletedStr)), totalDeletedColored,
		strings.Repeat(" ", netWidth-len(totalNetStr)), totalNetColored,
	)

	return nil
}

func formatNet(net int) string {
	if net > 0 {
		return fmt.Sprintf("+%d", net)
	}
	return strconv.Itoa(net)
}
