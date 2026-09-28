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

// TreeFormatter formats a model.TreeReport into a hierarchical tree view with Unicode branch lines.
type TreeFormatter struct {
	NoColor     bool
	ShowPercent bool
	ShowGraph   bool
	MaxGraphLen int
	MaxDepth    int // 0 means unlimited
}

// NewTreeFormatter creates a new TreeFormatter with default options.
func NewTreeFormatter(noColor bool) *TreeFormatter {
	return &TreeFormatter{
		NoColor:     noColor,
		MaxGraphLen: 20,
	}
}

// NewTreeFormatterWithOptions creates a new TreeFormatter with specified options.
func NewTreeFormatterWithOptions(noColor, showPercent, showGraph bool, maxDepth int) *TreeFormatter {
	return &TreeFormatter{
		NoColor:     noColor,
		ShowPercent: showPercent,
		ShowGraph:   showGraph,
		MaxGraphLen: 20,
		MaxDepth:    maxDepth,
	}
}

func (f *TreeFormatter) shouldUseColor(w io.Writer) bool {
	if f.NoColor {
		return false
	}
	if file, ok := w.(*os.File); ok {
		fd := file.Fd()
		return isatty.IsTerminal(fd) || isatty.IsCygwinTerminal(fd)
	}
	return false
}

type treeRow struct {
	treePrefix string
	name       string
	node       *model.TreeNode
}

// Format renders the tree report to the given writer.
func (f *TreeFormatter) Format(w io.Writer, report *model.TreeReport) error {
	useColor := f.shouldUseColor(w)

	maxGraphLen := f.MaxGraphLen
	if maxGraphLen <= 0 {
		maxGraphLen = 20
	}

	// 1. Output Summary Block
	target := report.Target
	if target == "" {
		target = "."
	}
	if _, err := fmt.Fprintf(w, "Target: %s\n", target); err != nil {
		return err
	}

	addedStr := fmt.Sprintf("+%d", report.Summary.TotalAdded)
	deletedStr := fmt.Sprintf("-%d", report.Summary.TotalDeleted)
	netStr := formatNet(report.Summary.Net)

	if useColor {
		if report.Summary.TotalAdded > 0 {
			addedStr = colorGreen + addedStr + colorReset
		}
		if report.Summary.TotalDeleted > 0 {
			deletedStr = colorRed + deletedStr + colorReset
		}
		if report.Summary.Net > 0 {
			netStr = colorGreen + netStr + colorReset
		} else if report.Summary.Net < 0 {
			netStr = colorRed + netStr + colorReset
		}
	}

	if _, err := fmt.Fprintf(w, "Files: %d  Added: %s  Deleted: %s  Net: %s\n\n",
		report.Summary.TotalFiles, addedStr, deletedStr, netStr); err != nil {
		return err
	}

	if report.Root == nil || len(report.Root.Children) == 0 {
		return nil
	}

	// 2. Flatten nodes according to tree hierarchy and MaxDepth
	var rows []treeRow
	f.flatten(report.Root, "", true, true, &rows)

	if len(rows) == 0 {
		return nil
	}

	// 3. Compute Column Widths
	maxPathLen := len("Directory")
	maxFilesLen := len("Files")
	maxAddedLen := len("Added")
	maxDeletedLen := len("Deleted")
	maxNetLen := len("Net")
	maxPercentLen := len("Percent")

	for _, r := range rows {
		displayPathLen := len([]rune(r.treePrefix + r.name))
		if displayPathLen > maxPathLen {
			maxPathLen = displayPathLen
		}

		filesLen := len(strconv.Itoa(r.node.Files))
		if filesLen > maxFilesLen {
			maxFilesLen = filesLen
		}

		aLen := len(strconv.Itoa(r.node.Added))
		if aLen > maxAddedLen {
			maxAddedLen = aLen
		}

		dLen := len(strconv.Itoa(r.node.Deleted))
		if dLen > maxDeletedLen {
			maxDeletedLen = dLen
		}

		nStr := formatNet(r.node.Net)
		if len(nStr) > maxNetLen {
			maxNetLen = len(nStr)
		}

		if f.ShowPercent {
			pStr := fmt.Sprintf("%.1f%%", r.node.Percent)
			if len(pStr) > maxPercentLen {
				maxPercentLen = len(pStr)
			}
		}
	}

	// 4. Print Header
	header := fmt.Sprintf("%-*s  %*s  %*s  %*s  %*s",
		maxPathLen, "Directory",
		maxFilesLen, "Files",
		maxAddedLen, "Added",
		maxDeletedLen, "Deleted",
		maxNetLen, "Net",
	)
	if f.ShowPercent {
		header += fmt.Sprintf("  %*s", maxPercentLen, "Percent")
	}
	if f.ShowGraph {
		header += "  Graph"
	}

	if _, err := fmt.Fprintln(w, header); err != nil {
		return err
	}

	// 5. Print Rows
	totalChanges := report.Summary.TotalAdded + report.Summary.TotalDeleted

	for _, r := range rows {
		fullPathCol := r.treePrefix + r.name
		pad := maxPathLen - len([]rune(fullPathCol))
		if pad < 0 {
			pad = 0
		}
		pathPart := fullPathCol + strings.Repeat(" ", pad)

		filesPart := fmt.Sprintf("%*d", maxFilesLen, r.node.Files)
		addedNum := fmt.Sprintf("%*d", maxAddedLen, r.node.Added)
		deletedNum := fmt.Sprintf("%*d", maxDeletedLen, r.node.Deleted)

		netVal := formatNet(r.node.Net)
		netPart := fmt.Sprintf("%*s", maxNetLen, netVal)

		if useColor {
			if r.node.Added > 0 {
				addedNum = colorGreen + addedNum + colorReset
			}
			if r.node.Deleted > 0 {
				deletedNum = colorRed + deletedNum + colorReset
			}
			if r.node.Net > 0 {
				netPart = colorGreen + netPart + colorReset
			} else if r.node.Net < 0 {
				netPart = colorRed + netPart + colorReset
			}
		}

		line := fmt.Sprintf("%s  %s  %s  %s  %s", pathPart, filesPart, addedNum, deletedNum, netPart)

		if f.ShowPercent {
			pStr := fmt.Sprintf("%.1f%%", r.node.Percent)
			line += fmt.Sprintf("  %*s", maxPercentLen, pStr)
		}

		if f.ShowGraph {
			graphBar := buildGraphBar(r.node.Added, r.node.Deleted, totalChanges, maxGraphLen, useColor)
			line += "  " + graphBar
		}

		if _, err := fmt.Fprintln(w, line); err != nil {
			return err
		}
	}

	return nil
}

func (f *TreeFormatter) flatten(node *model.TreeNode, prefix string, isLast bool, isRoot bool, rows *[]treeRow) {
	if isRoot {
		*rows = append(*rows, treeRow{
			treePrefix: "",
			name:       node.Name,
			node:       node,
		})
	} else {
		branch := "├── "
		if isLast {
			branch = "└── "
		}
		*rows = append(*rows, treeRow{
			treePrefix: prefix + branch,
			name:       node.Name,
			node:       node,
		})
	}

	if f.MaxDepth > 0 && node.Depth >= f.MaxDepth {
		return
	}

	// Prepare child prefix
	childPrefix := prefix
	if !isRoot {
		if isLast {
			childPrefix += "    "
		} else {
			childPrefix += "│   "
		}
	}

	for i, child := range node.Children {
		isLastChild := (i == len(node.Children)-1)
		f.flatten(child, childPrefix, isLastChild, false, rows)
	}
}
