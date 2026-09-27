package formatter

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/sakurahilljp/git-dirstat/pkg/model"
)

// MarkdownFormatter formats a model.Report as a GitHub Flavored Markdown (GFM) table.
type MarkdownFormatter struct {
	ShowPercent bool
	ShowGraph   bool
	MaxGraphLen int
}

// NewMarkdownFormatter creates a new MarkdownFormatter with default options.
func NewMarkdownFormatter() *MarkdownFormatter {
	return &MarkdownFormatter{
		MaxGraphLen: 20,
	}
}

// NewMarkdownFormatterWithOptions creates a new MarkdownFormatter with specified percent and graph options.
func NewMarkdownFormatterWithOptions(showPercent, showGraph bool) *MarkdownFormatter {
	return &MarkdownFormatter{
		ShowPercent: showPercent,
		ShowGraph:   showGraph,
		MaxGraphLen: 20,
	}
}

// Format writes the report in GitHub Flavored Markdown table syntax.
func (f *MarkdownFormatter) Format(w io.Writer, report *model.Report) error {
	maxGraphLen := f.MaxGraphLen
	if maxGraphLen <= 0 {
		maxGraphLen = 20
	}

	totalChanges := report.Summary.TotalAdded + report.Summary.TotalDeleted

	writeLine := func(format string, a ...any) error {
		_, err := fmt.Fprintf(w, format, a...)
		return err
	}

	// 1. Target line: **Target:** `<target>` (Depth: <depth>)
	cleanTarget := strings.ReplaceAll(report.Target, "`", "")
	if err := writeLine("**Target:** `%s` (Depth: %d)\n\n", cleanTarget, report.Depth); err != nil {
		return err
	}

	// 2. Table Headers
	headers := []string{"Directory", "Files", "Added", "Deleted", "Net"}
	alignments := []string{":---", "---:", "---:", "---:", "---:"}

	if f.ShowPercent {
		headers = append(headers, "Percent")
		alignments = append(alignments, "---:")
	}
	if f.ShowGraph {
		headers = append(headers, "Graph")
		alignments = append(alignments, ":---")
	}

	if err := writeLine("| %s |\n", strings.Join(headers, " | ")); err != nil {
		return err
	}
	if err := writeLine("| %s |\n", strings.Join(alignments, " | ")); err != nil {
		return err
	}

	// 3. Data rows
	for _, e := range report.Entries {
		dirStr := FormatEntryPath(e, report.Target)
		cleanDir := strings.ReplaceAll(dirStr, "`", "")
		cols := []string{
			fmt.Sprintf("`%s`", cleanDir),
			strconv.Itoa(e.Files),
			strconv.Itoa(e.Added),
			strconv.Itoa(e.Deleted),
			formatNet(e.Net),
		}

		if f.ShowPercent {
			cols = append(cols, fmt.Sprintf("%.1f%%", e.Percent))
		}
		if f.ShowGraph {
			graphBar := buildGraphBar(e.Added, e.Deleted, totalChanges, maxGraphLen, false)
			if graphBar != "" {
				cols = append(cols, fmt.Sprintf("`%s`", graphBar))
			} else {
				cols = append(cols, "")
			}
		}

		if err := writeLine("| %s |\n", strings.Join(cols, " | ")); err != nil {
			return err
		}
	}

	// 4. TOTAL row
	totalCols := []string{
		"**TOTAL**",
		fmt.Sprintf("**%d**", report.Summary.TotalFiles),
		fmt.Sprintf("**%d**", report.Summary.TotalAdded),
		fmt.Sprintf("**%d**", report.Summary.TotalDeleted),
		fmt.Sprintf("**%s**", formatNet(report.Summary.Net)),
	}

	if f.ShowPercent {
		totalPercent := "0.0%"
		if totalChanges > 0 {
			totalPercent = "100.0%"
		}
		totalCols = append(totalCols, fmt.Sprintf("**%s**", totalPercent))
	}
	if f.ShowGraph {
		totalBar := buildGraphBar(report.Summary.TotalAdded, report.Summary.TotalDeleted, totalChanges, maxGraphLen, false)
		if totalBar != "" {
			totalCols = append(totalCols, fmt.Sprintf("`%s`", totalBar))
		} else {
			totalCols = append(totalCols, "")
		}
	}

	return writeLine("| %s |\n", strings.Join(totalCols, " | "))
}
