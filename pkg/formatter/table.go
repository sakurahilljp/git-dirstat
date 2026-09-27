package formatter

import (
	"fmt"
	"io"
	"math"
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
	NoColor     bool
	ShowPercent bool
	ShowGraph   bool
	MaxGraphLen int
}

func NewTableFormatter(noColor bool) *TableFormatter {
	return &TableFormatter{
		NoColor:     noColor,
		MaxGraphLen: 20,
	}
}

func NewTableFormatterWithOptions(noColor, showPercent, showGraph bool) *TableFormatter {
	return &TableFormatter{
		NoColor:     noColor,
		ShowPercent: showPercent,
		ShowGraph:   showGraph,
		MaxGraphLen: 20,
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

	maxGraphLen := f.MaxGraphLen
	if maxGraphLen <= 0 {
		maxGraphLen = 20
	}
	graphWidth := maxGraphLen
	if graphWidth < 5 {
		graphWidth = 5
	}
	percentWidth := 8

	totalChanges := report.Summary.TotalAdded + report.Summary.TotalDeleted

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
		percent string
		graph   string
	}

	var rows []rowData
	for _, e := range report.Entries {
		dirStr := FormatEntryPath(e, report.Target)
		filesStr := strconv.Itoa(e.Files)
		addedStr := strconv.Itoa(e.Added)
		deletedStr := strconv.Itoa(e.Deleted)
		netStr := formatNet(e.Net)
		percentStr := fmt.Sprintf("%5.1f%%", e.Percent)
		graphStr := buildGraphBar(e.Added, e.Deleted, totalChanges, maxGraphLen, useColor)

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
			percent: percentStr,
			graph:   graphStr,
		})
	}

	totalFilesStr := strconv.Itoa(report.Summary.TotalFiles)
	totalAddedStr := strconv.Itoa(report.Summary.TotalAdded)
	totalDeletedStr := strconv.Itoa(report.Summary.TotalDeleted)
	totalNetStr := formatNet(report.Summary.Net)

	totalPercentStr := "  0.0%"
	if totalChanges > 0 {
		totalPercentStr = "100.0%"
	}
	totalGraphStr := buildGraphBar(report.Summary.TotalAdded, report.Summary.TotalDeleted, totalChanges, maxGraphLen, useColor)

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
	if f.ShowPercent {
		totalWidth += 1 + percentWidth
	}
	if f.ShowGraph {
		totalWidth += 1 + graphWidth
	}
	sep := strings.Repeat("-", totalWidth)

	// Target line
	fmt.Fprintf(w, "Target: %s (Depth: %d)\n\n", report.Target, report.Depth)

	// Header line
	var header strings.Builder
	header.WriteString(fmt.Sprintf("%-*s %*s %*s %*s %*s",
		dirWidth, "Directory",
		filesWidth, "Files",
		addedWidth, "Added",
		deletedWidth, "Deleted",
		netWidth, "Net",
	))
	if f.ShowPercent {
		header.WriteString(fmt.Sprintf(" %*s", percentWidth, "Percent"))
	}
	if f.ShowGraph {
		header.WriteString(" Graph")
	}
	header.WriteString("\n")
	w.Write([]byte(header.String()))
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

		var rowLine strings.Builder
		rowLine.WriteString(fmt.Sprintf("%-*s %*s %s%s %s%s %s%s",
			dirWidth, r.dir,
			filesWidth, r.files,
			strings.Repeat(" ", addedWidth-len(r.added)), addedColored,
			strings.Repeat(" ", deletedWidth-len(r.deleted)), deletedColored,
			strings.Repeat(" ", netWidth-len(r.net)), netColored,
		))
		if f.ShowPercent {
			rowLine.WriteString(fmt.Sprintf(" %*s", percentWidth, r.percent))
		}
		if f.ShowGraph {
			rowLine.WriteString(" ")
			rowLine.WriteString(r.graph)
		}
		rowLine.WriteString("\n")
		w.Write([]byte(rowLine.String()))
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

	var totalLine strings.Builder
	totalLine.WriteString(fmt.Sprintf("%-*s %*s %s%s %s%s %s%s",
		dirWidth, "TOTAL",
		filesWidth, totalFilesStr,
		strings.Repeat(" ", addedWidth-len(totalAddedStr)), totalAddedColored,
		strings.Repeat(" ", deletedWidth-len(totalDeletedStr)), totalDeletedColored,
		strings.Repeat(" ", netWidth-len(totalNetStr)), totalNetColored,
	))
	if f.ShowPercent {
		totalLine.WriteString(fmt.Sprintf(" %*s", percentWidth, totalPercentStr))
	}
	if f.ShowGraph {
		totalLine.WriteString(" ")
		totalLine.WriteString(totalGraphStr)
	}
	totalLine.WriteString("\n")
	w.Write([]byte(totalLine.String()))

	return nil
}

func buildGraphBar(added, deleted, totalChanges, maxLen int, useColor bool) string {
	rowChanges := added + deleted
	if totalChanges <= 0 || rowChanges <= 0 || maxLen <= 0 {
		return ""
	}

	barLen := int(math.Round(float64(rowChanges) / float64(totalChanges) * float64(maxLen)))
	if barLen == 0 && rowChanges > 0 {
		barLen = 1
	}
	if barLen > maxLen {
		barLen = maxLen
	}

	var plusCount, minusCount int
	if added > 0 && deleted == 0 {
		plusCount = barLen
		minusCount = 0
	} else if added == 0 && deleted > 0 {
		plusCount = 0
		minusCount = barLen
	} else {
		plusCount = int(math.Round(float64(added) / float64(rowChanges) * float64(barLen)))
		minusCount = barLen - plusCount

		// Ensure both + and - appear if both added and deleted > 0 and barLen >= 2
		if barLen >= 2 {
			if plusCount == 0 && added > 0 {
				plusCount = 1
				minusCount = barLen - 1
			} else if minusCount == 0 && deleted > 0 {
				minusCount = 1
				plusCount = barLen - 1
			}
		}
	}

	if useColor {
		var sb strings.Builder
		if plusCount > 0 {
			sb.WriteString(colorGreen)
			sb.WriteString(strings.Repeat("+", plusCount))
			sb.WriteString(colorReset)
		}
		if minusCount > 0 {
			sb.WriteString(colorRed)
			sb.WriteString(strings.Repeat("-", minusCount))
			sb.WriteString(colorReset)
		}
		return sb.String()
	}

	return strings.Repeat("+", plusCount) + strings.Repeat("-", minusCount)
}

func formatNet(net int) string {
	if net > 0 {
		return fmt.Sprintf("+%d", net)
	}
	return strconv.Itoa(net)
}
