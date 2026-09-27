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

type ChurnTableFormatter struct {
	NoColor     bool
	ShowPercent bool
	ShowGraph   bool
	MaxGraphLen int
}

func NewChurnTableFormatter(noColor, showPercent, showGraph bool) *ChurnTableFormatter {
	return &ChurnTableFormatter{
		NoColor:     noColor,
		ShowPercent: showPercent,
		ShowGraph:   showGraph,
		MaxGraphLen: 20,
	}
}

func (f *ChurnTableFormatter) shouldUseColor(w io.Writer) bool {
	if f.NoColor {
		return false
	}
	if file, ok := w.(*os.File); ok {
		fd := file.Fd()
		return isatty.IsTerminal(fd) || isatty.IsCygwinTerminal(fd)
	}
	return false
}

type churnTableRow struct {
	dir     string
	commits string
	files   string
	added   string
	deleted string
	churn   string
	percent string
	graph   string
}

func (f *ChurnTableFormatter) FormatChurn(w io.Writer, report *model.ChurnReport) error {
	useColor := f.shouldUseColor(w)

	maxGraphLen := f.MaxGraphLen
	if maxGraphLen <= 0 {
		maxGraphLen = 20
	}

	dirWidth := len("Directory")
	commitsWidth := len("Commits")
	filesWidth := len("Files")
	addedWidth := len("Added")
	deletedWidth := len("Deleted")
	churnWidth := len("Churn")
	percentWidth := len("Percent")

	rows := make([]churnTableRow, len(report.Entries))
	for i, entry := range report.Entries {
		dir := FormatChurnEntryPath(entry, report.Target)
		commitsStr := strconv.Itoa(entry.Commits)
		filesStr := strconv.Itoa(entry.Files)
		addedStr := fmt.Sprintf("+%d", entry.Added)
		deletedStr := fmt.Sprintf("-%d", entry.Deleted)
		churnStr := strconv.Itoa(entry.Churn)
		percentStr := fmt.Sprintf("%.1f%%", entry.Percent)

		graphStr := ""
		if f.ShowGraph {
			graphStr = buildGraphBar(entry.Added, entry.Deleted, report.Summary.TotalChurn, maxGraphLen, useColor)
		}

		if len(dir) > dirWidth {
			dirWidth = len(dir)
		}
		if len(commitsStr) > commitsWidth {
			commitsWidth = len(commitsStr)
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
		if len(churnStr) > churnWidth {
			churnWidth = len(churnStr)
		}
		if len(percentStr) > percentWidth {
			percentWidth = len(percentStr)
		}

		rows[i] = churnTableRow{
			dir:     dir,
			commits: commitsStr,
			files:   filesStr,
			added:   addedStr,
			deleted: deletedStr,
			churn:   churnStr,
			percent: percentStr,
			graph:   graphStr,
		}
	}

	totalCommitsStr := strconv.Itoa(report.Summary.TotalCommits)
	totalFilesStr := strconv.Itoa(report.Summary.TotalFiles)
	totalAddedStr := fmt.Sprintf("+%d", report.Summary.TotalAdded)
	totalDeletedStr := fmt.Sprintf("-%d", report.Summary.TotalDeleted)
	totalChurnStr := strconv.Itoa(report.Summary.TotalChurn)
	totalPercentStr := "100.0%"

	if len("TOTAL") > dirWidth {
		dirWidth = len("TOTAL")
	}
	if len(totalCommitsStr) > commitsWidth {
		commitsWidth = len(totalCommitsStr)
	}
	if len(totalFilesStr) > filesWidth {
		filesWidth = len(totalFilesStr)
	}
	if len(totalAddedStr) > addedWidth {
		addedWidth = len(totalAddedStr)
	}
	if len(totalDeletedStr) > deletedWidth {
		deletedWidth = len(totalDeletedStr)
	}
	if len(totalChurnStr) > churnWidth {
		churnWidth = len(totalChurnStr)
	}
	if len(totalPercentStr) > percentWidth {
		percentWidth = len(totalPercentStr)
	}

	totalGraphStr := ""
	if f.ShowGraph {
		totalGraphStr = buildGraphBar(report.Summary.TotalAdded, report.Summary.TotalDeleted, report.Summary.TotalChurn, maxGraphLen, useColor)
	}

	// Header line
	var headerLine strings.Builder
	headerLine.WriteString(fmt.Sprintf("%-*s %*s %*s %*s %*s %*s",
		dirWidth, "Directory",
		commitsWidth, "Commits",
		filesWidth, "Files",
		addedWidth, "Added",
		deletedWidth, "Deleted",
		churnWidth, "Churn",
	))
	if f.ShowPercent {
		headerLine.WriteString(fmt.Sprintf(" %*s", percentWidth, "Percent"))
	}
	if f.ShowGraph {
		headerLine.WriteString(" Graph")
	}
	fmt.Fprintln(w, headerLine.String())

	// Separator line
	totalWidth := dirWidth + 1 + commitsWidth + 1 + filesWidth + 1 + addedWidth + 1 + deletedWidth + 1 + churnWidth
	if f.ShowPercent {
		totalWidth += 1 + percentWidth
	}
	if f.ShowGraph {
		totalWidth += 1 + maxGraphLen
	}
	sep := strings.Repeat("-", totalWidth)
	fmt.Fprintln(w, sep)

	// Data rows
	for _, r := range rows {
		addedColored := r.added
		deletedColored := r.deleted

		if useColor {
			if r.added != "+0" {
				addedColored = colorGreen + r.added + colorReset
			}
			if r.deleted != "-0" {
				deletedColored = colorRed + r.deleted + colorReset
			}
		}

		var rowLine strings.Builder
		rowLine.WriteString(fmt.Sprintf("%-*s %*s %*s %s%s %s%s %*s",
			dirWidth, r.dir,
			commitsWidth, r.commits,
			filesWidth, r.files,
			strings.Repeat(" ", addedWidth-len(r.added)), addedColored,
			strings.Repeat(" ", deletedWidth-len(r.deleted)), deletedColored,
			churnWidth, r.churn,
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
	if useColor {
		if report.Summary.TotalAdded != 0 {
			totalAddedColored = colorGreen + totalAddedStr + colorReset
		}
		if report.Summary.TotalDeleted != 0 {
			totalDeletedColored = colorRed + totalDeletedStr + colorReset
		}
	}

	var totalLine strings.Builder
	totalLine.WriteString(fmt.Sprintf("%-*s %*s %*s %s%s %s%s %*s",
		dirWidth, "TOTAL",
		commitsWidth, totalCommitsStr,
		filesWidth, totalFilesStr,
		strings.Repeat(" ", addedWidth-len(totalAddedStr)), totalAddedColored,
		strings.Repeat(" ", deletedWidth-len(totalDeletedStr)), totalDeletedColored,
		churnWidth, totalChurnStr,
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
