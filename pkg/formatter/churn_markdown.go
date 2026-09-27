package formatter

import (
	"fmt"
	"io"
	"strings"

	"github.com/sakurahilljp/git-dirstat/pkg/model"
)

type ChurnMarkdownFormatter struct {
	ShowPercent bool
	ShowGraph   bool
	MaxGraphLen int
}

func NewChurnMarkdownFormatterWithOptions(showPercent, showGraph bool) *ChurnMarkdownFormatter {
	return &ChurnMarkdownFormatter{
		ShowPercent: showPercent,
		ShowGraph:   showGraph,
		MaxGraphLen: 20,
	}
}

func (f *ChurnMarkdownFormatter) FormatChurn(w io.Writer, report *model.ChurnReport) error {
	maxGraphLen := f.MaxGraphLen
	if maxGraphLen <= 0 {
		maxGraphLen = 20
	}

	var headers []string
	var aligns []string

	headers = append(headers, "Directory", "Commits", "Files", "Added", "Deleted", "Churn")
	aligns = append(aligns, ":---", "---:", "---:", "---:", "---:", "---:")

	if f.ShowPercent {
		headers = append(headers, "Percent")
		aligns = append(aligns, "---:")
	}
	if f.ShowGraph {
		headers = append(headers, "Graph")
		aligns = append(aligns, ":---")
	}

	fmt.Fprintf(w, "| %s |\n", strings.Join(headers, " | "))
	fmt.Fprintf(w, "| %s |\n", strings.Join(aligns, " | "))

	// Data rows
	for _, entry := range report.Entries {
		dir := FormatChurnEntryPath(entry, report.Target)
		var cols []string
		cols = append(cols,
			fmt.Sprintf("`%s`", dir),
			fmt.Sprintf("%d", entry.Commits),
			fmt.Sprintf("%d", entry.Files),
			fmt.Sprintf("+%d", entry.Added),
			fmt.Sprintf("-%d", entry.Deleted),
			fmt.Sprintf("%d", entry.Churn),
		)

		if f.ShowPercent {
			cols = append(cols, fmt.Sprintf("%.1f%%", entry.Percent))
		}
		if f.ShowGraph {
			graphBar := buildGraphBar(entry.Added, entry.Deleted, report.Summary.TotalChurn, maxGraphLen, false)
			if graphBar == "" {
				cols = append(cols, "")
			} else {
				cols = append(cols, fmt.Sprintf("`%s`", graphBar))
			}
		}

		fmt.Fprintf(w, "| %s |\n", strings.Join(cols, " | "))
	}

	// Total row
	var totalCols []string
	totalCols = append(totalCols,
		"**TOTAL**",
		fmt.Sprintf("**%d**", report.Summary.TotalCommits),
		fmt.Sprintf("**%d**", report.Summary.TotalFiles),
		fmt.Sprintf("**+%d**", report.Summary.TotalAdded),
		fmt.Sprintf("**-%d**", report.Summary.TotalDeleted),
		fmt.Sprintf("**%d**", report.Summary.TotalChurn),
	)
	if f.ShowPercent {
		totalCols = append(totalCols, "**100.0%**")
	}
	if f.ShowGraph {
		totalGraph := buildGraphBar(report.Summary.TotalAdded, report.Summary.TotalDeleted, report.Summary.TotalChurn, maxGraphLen, false)
		if totalGraph == "" {
			totalCols = append(totalCols, "")
		} else {
			totalCols = append(totalCols, fmt.Sprintf("`%s`", totalGraph))
		}
	}

	fmt.Fprintf(w, "| %s |\n", strings.Join(totalCols, " | "))
	return nil
}
