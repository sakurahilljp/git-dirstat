package tui

import (
	"fmt"
	"io"
	"math"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sakurahilljp/git-dirstat/pkg/model"
)

// Options holds runtime options for the interactive TUI.
type Options struct {
	ShowPercent bool
	ShowGraph   bool
	NoColor     bool
	SortField   string
	Reverse     bool
	Stdin       io.Reader
	Stdout      io.Writer
}

type flatNode struct {
	node   *model.TreeNode
	prefix string
	depth  int
}

// AppModel is the Bubbletea model for git-dirstat TUI.
type AppModel struct {
	report       *model.TreeReport
	opts         Options
	width        int
	height       int
	cursor       int
	visibleNodes []flatNode
	sortIndex    int
	sortFields   []string
	filterInput  textinput.Model
	isFiltering  bool
	filterQuery  string
	showHelp     bool
	scrollOffset int
}

var defaultSortFields = []string{
	model.SortAdded,
	model.SortDeleted,
	model.SortNet,
	model.SortPercent,
	model.SortFiles,
	model.SortPath,
}

// NewAppModel creates an initialized AppModel.
func NewAppModel(report *model.TreeReport, opts Options) *AppModel {
	ti := textinput.New()
	ti.Placeholder = "filter by path..."
	ti.Prompt = "/ "

	initialSortField := opts.SortField
	if initialSortField == "" {
		initialSortField = model.SortAdded
	}

	sortIdx := 0
	for i, f := range defaultSortFields {
		if f == initialSortField {
			sortIdx = i
			break
		}
	}

	m := &AppModel{
		report:      report,
		opts:        opts,
		width:       100,
		height:      30,
		cursor:      0,
		sortIndex:   sortIdx,
		sortFields:  defaultSortFields,
		filterInput: ti,
		showHelp:    false,
	}

	m.rebuildVisibleNodes()
	return m
}

func (m *AppModel) Init() tea.Cmd {
	return nil
}

func (m *AppModel) rebuildVisibleNodes() {
	m.visibleNodes = nil
	if m.report == nil || m.report.Root == nil {
		return
	}

	var collect func(node *model.TreeNode, prefix string, isLast bool, isRoot bool)
	collect = func(node *model.TreeNode, prefix string, isLast bool, isRoot bool) {
		if !nodeMatchesQuery(node, m.filterQuery) {
			return
		}

		displayPrefix := ""
		if !isRoot {
			branch := "├── "
			if isLast {
				branch = "└── "
			}
			displayPrefix = prefix + branch
		}

		m.visibleNodes = append(m.visibleNodes, flatNode{
			node:   node,
			prefix: displayPrefix,
			depth:  node.Depth,
		})

		expanded := node.Expanded
		if m.filterQuery != "" {
			expanded = true // Auto-expand matching descendants in filter mode
		}

		if !node.IsDir || !expanded {
			return
		}

		childPrefix := prefix
		if !isRoot {
			if isLast {
				childPrefix += "    "
			} else {
				childPrefix += "│   "
			}
		}

		var matchingChildren []*model.TreeNode
		for _, child := range node.Children {
			if nodeMatchesQuery(child, m.filterQuery) {
				matchingChildren = append(matchingChildren, child)
			}
		}

		for i, child := range matchingChildren {
			isLastChild := (i == len(matchingChildren)-1)
			collect(child, childPrefix, isLastChild, false)
		}
	}

	collect(m.report.Root, "", true, true)

	if m.cursor >= len(m.visibleNodes) {
		m.cursor = len(m.visibleNodes) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

func (m *AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		if m.isFiltering {
			switch msg.String() {
			case "enter", "esc":
				m.isFiltering = false
				m.filterInput.Blur()
				m.filterQuery = m.filterInput.Value()
				m.rebuildVisibleNodes()
				return m, nil
			default:
				var cmd tea.Cmd
				m.filterInput, cmd = m.filterInput.Update(msg)
				m.filterQuery = m.filterInput.Value()
				m.rebuildVisibleNodes()
				return m, cmd
			}
		}

		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit

		case "?":
			m.showHelp = !m.showHelp
			return m, nil

		case "esc":
			if m.showHelp {
				m.showHelp = false
				return m, nil
			}
			if m.filterQuery != "" {
				m.filterQuery = ""
				m.filterInput.SetValue("")
				m.rebuildVisibleNodes()
				return m, nil
			}

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				m.adjustScroll()
			}

		case "down", "j":
			if m.cursor < len(m.visibleNodes)-1 {
				m.cursor++
				m.adjustScroll()
			}

		case "enter", " ", "right", "l":
			if len(m.visibleNodes) > 0 {
				curr := m.visibleNodes[m.cursor].node
				if curr.IsDir {
					curr.Expanded = !curr.Expanded
					m.rebuildVisibleNodes()
				}
			}

		case "left", "h":
			if len(m.visibleNodes) > 0 {
				curr := m.visibleNodes[m.cursor].node
				if curr.IsDir && curr.Expanded {
					curr.Expanded = false
					m.rebuildVisibleNodes()
				} else if m.cursor > 0 {
					startDepth := m.visibleNodes[m.cursor].depth
					for i := m.cursor - 1; i >= 0; i-- {
						if m.visibleNodes[i].depth < startDepth {
							m.cursor = i
							m.adjustScroll()
							break
						}
					}
				}
			}

		case "o":
			m.setAllExpanded(m.report.Root, true)
			m.rebuildVisibleNodes()

		case "O":
			m.setAllExpanded(m.report.Root, false)
			if m.report.Root != nil {
				m.report.Root.Expanded = true // Keep root open
			}
			m.rebuildVisibleNodes()

		case "s":
			m.sortIndex = (m.sortIndex + 1) % len(m.sortFields)
			m.opts.SortField = m.sortFields[m.sortIndex]
			m.resortTree(m.report.Root)
			m.rebuildVisibleNodes()

		case "p":
			m.opts.ShowPercent = !m.opts.ShowPercent

		case "g":
			m.opts.ShowGraph = !m.opts.ShowGraph

		case "/":
			m.isFiltering = true
			m.filterInput.Focus()
			return m, textinput.Blink
		}
	}

	return m, nil
}

func (m *AppModel) adjustScroll() {
	contentHeight := m.height - 7 // Header, footer, margins
	if contentHeight <= 0 {
		contentHeight = 10
	}
	if m.cursor < m.scrollOffset {
		m.scrollOffset = m.cursor
	} else if m.cursor >= m.scrollOffset+contentHeight {
		m.scrollOffset = m.cursor - contentHeight + 1
	}
}

func (m *AppModel) setAllExpanded(node *model.TreeNode, expanded bool) {
	if node == nil {
		return
	}
	if node.IsDir {
		node.Expanded = expanded
	}
	for _, child := range node.Children {
		m.setAllExpanded(child, expanded)
	}
}

func (m *AppModel) resortTree(node *model.TreeNode) {
	if node == nil {
		return
	}
	sortTreeChildren(node.Children, m.opts.SortField, m.opts.Reverse)
	for _, child := range node.Children {
		m.resortTree(child)
	}
}

func sortTreeChildren(nodes []*model.TreeNode, sortField string, reverse bool) {
	sort.Slice(nodes, func(i, j int) bool {
		a, b := nodes[i], nodes[j]
		if a.IsDir != b.IsDir {
			return a.IsDir
		}
		var cmp bool
		switch sortField {
		case model.SortPath:
			cmp = a.Name < b.Name
		case model.SortFiles:
			if a.Files != b.Files {
				cmp = a.Files > b.Files
			} else {
				cmp = a.Name < b.Name
			}
		case model.SortDeleted:
			if a.Deleted != b.Deleted {
				cmp = a.Deleted > b.Deleted
			} else {
				cmp = a.Name < b.Name
			}
		case model.SortNet:
			if a.Net != b.Net {
				cmp = a.Net > b.Net
			} else {
				cmp = a.Name < b.Name
			}
		case model.SortPercent:
			aChanges := a.Added + a.Deleted
			bChanges := b.Added + b.Deleted
			if aChanges != bChanges {
				cmp = aChanges > bChanges
			} else {
				cmp = a.Name < b.Name
			}
		case model.SortAdded:
			fallthrough
		default:
			if a.Added != b.Added {
				cmp = a.Added > b.Added
			} else {
				cmp = a.Name < b.Name
			}
		}
		if reverse {
			return !cmp
		}
		return cmp
	})
}

func (m *AppModel) View() string {
	if m.width <= 0 || m.height <= 0 {
		return "Initializing git-dirstat TUI..."
	}

	// 1. Header
	header := m.renderHeader()

	// 2. Main Content (2 Panes)
	mainContent := m.renderMainPanes()

	// 3. Footer / Status bar
	footer := m.renderFooter()

	baseView := lipgloss.JoinVertical(lipgloss.Left, header, mainContent, footer)

	// 4. Help Modal Overlay
	if m.showHelp {
		return m.renderHelpOverlay(baseView)
	}

	return baseView
}

func (m *AppModel) renderHeader() string {
	title := titleStyle.Render("git-dirstat TUI")
	target := m.report.Target
	if target == "" {
		target = "."
	}

	sum := m.report.Summary
	stats := fmt.Sprintf("Target: %s  |  Files: %d  Added: +%d  Deleted: -%d  Net: %+d",
		target, sum.TotalFiles, sum.TotalAdded, sum.TotalDeleted, sum.Net)
	info := headerInfoStyle.Render(stats)

	return lipgloss.JoinHorizontal(lipgloss.Center, title, "  ", info) + "\n"
}

func (m *AppModel) renderMainPanes() string {
	contentHeight := m.height - 6
	if contentHeight < 5 {
		contentHeight = 5
	}

	// Two pane layout: Left = Tree, Right = File Details
	// If width is tight, tree gets full width
	if m.width < 80 {
		treeWidth := m.width - 4
		treeContent := m.renderTreePane(treeWidth, contentHeight)
		return paneStyle.Width(treeWidth).Height(contentHeight).Render(treeContent)
	}

	leftWidth := (m.width * 60) / 100
	rightWidth := m.width - leftWidth - 5

	treeContent := m.renderTreePane(leftWidth-4, contentHeight)
	detailContent := m.renderDetailPane(rightWidth-4, contentHeight)

	leftPane := focusedPaneStyle.Width(leftWidth).Height(contentHeight).Render(treeContent)
	rightPane := paneStyle.Width(rightWidth).Height(contentHeight).Render(detailContent)

	return lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)
}

func (m *AppModel) renderTreePane(width, height int) string {
	var lines []string
	title := paneTitleStyle.Render("Directory Tree")
	lines = append(lines, title)

	if len(m.visibleNodes) == 0 {
		lines = append(lines, "  (No matching directories or files)")
		return strings.Join(lines, "\n")
	}

	endIdx := m.scrollOffset + height - 2
	if endIdx > len(m.visibleNodes) {
		endIdx = len(m.visibleNodes)
	}

	totalChanges := m.report.Summary.TotalAdded + m.report.Summary.TotalDeleted

	for i := m.scrollOffset; i < endIdx; i++ {
		fn := m.visibleNodes[i]
		n := fn.node

		icon := "  "
		if n.IsDir {
			if n.Expanded {
				icon = "▼ "
			} else {
				icon = "▶ "
			}
		}

		namePart := n.Name
		if n.IsDir {
			namePart = dirStyle.Render(n.Name)
		} else {
			namePart = fileStyle.Render(n.Name)
		}

		diffPart := fmt.Sprintf("+%d/-%d", n.Added, n.Deleted)
		if n.Added > 0 {
			diffPart = addedStyle.Render(fmt.Sprintf("+%d", n.Added)) + "/" + deletedStyle.Render(fmt.Sprintf("-%d", n.Deleted))
		}

		pctPart := ""
		if m.opts.ShowPercent {
			pctPart = fmt.Sprintf(" %.1f%%", n.Percent)
		}

		graphPart := ""
		if m.opts.ShowGraph && totalChanges > 0 {
			barLen := int(math.Round(float64(n.Added+n.Deleted) / float64(totalChanges) * 10))
			if barLen == 0 && (n.Added+n.Deleted) > 0 {
				barLen = 1
			}
			graphPart = " " + strings.Repeat("■", barLen)
		}

		rowText := fmt.Sprintf("%s%s%s (%d files, %s%s%s)",
			fn.prefix, icon, namePart, n.Files, diffPart, pctPart, graphPart)

		if i == m.cursor {
			rowText = cursorStyle.Render("> " + rowText)
		} else {
			rowText = "  " + rowText
		}

		lines = append(lines, rowText)
	}

	return strings.Join(lines, "\n")
}

func (m *AppModel) renderDetailPane(width, height int) string {
	var lines []string
	lines = append(lines, detailHeaderStyle.Render("Details / Files"))

	if len(m.visibleNodes) == 0 || m.cursor >= len(m.visibleNodes) {
		lines = append(lines, "  (Select an entry to view details)")
		return strings.Join(lines, "\n")
	}

	curr := m.visibleNodes[m.cursor].node
	lines = append(lines, fmt.Sprintf("Path: %s", curr.Path))
	lines = append(lines, fmt.Sprintf("Files: %d  |  Net: %+d", curr.Files, curr.Net))
	lines = append(lines, strings.Repeat("─", width))

	if len(curr.FileDiffs) == 0 {
		if curr.IsDir {
			lines = append(lines, "  (No direct files in this directory)")
		} else {
			lines = append(lines, fmt.Sprintf("  File diff: +%d / -%d", curr.Added, curr.Deleted))
		}
	} else {
		for _, f := range curr.FileDiffs {
			fName := f.Path
			diffInfo := fmt.Sprintf("+%d / -%d", f.Added, f.Deleted)
			lines = append(lines, fmt.Sprintf("• %s (%s)", fName, diffInfo))
			if len(lines) >= height-1 {
				lines = append(lines, "  ... (more files truncated)")
				break
			}
		}
	}

	return strings.Join(lines, "\n")
}

func (m *AppModel) renderFooter() string {
	if m.isFiltering {
		return m.filterInput.View()
	}

	sortText := fmt.Sprintf("Sort: %s", m.opts.SortField)
	keys := "[↑/↓/j/k] Move  [Space/Enter] Toggle  [o/O] Expand/Collapse  [s] Sort  [p] %  [g] Graph  [/] Filter  [?] Help  [q] Quit"

	bar := fmt.Sprintf(" %s  |  %s", sortText, keys)
	return statusBarStyle.Width(m.width).Render(bar)
}

func (m *AppModel) renderHelpOverlay(background string) string {
	helpText := `git-dirstat Interactive TUI Help

Navigation:
  ↑, k          Move cursor up
  ↓, j          Move cursor down
  Space, Enter  Expand / collapse directory
  →, l          Expand directory
  ←, h          Collapse directory or step to parent
  o             Expand all directories
  O             Collapse all directories (except root)

Display & Options:
  s             Cycle sort field (added, deleted, net, percent, files, path)
  p             Toggle percentage column
  g             Toggle bar graph column
  /             Filter tree by path
  Esc           Clear filter / close modal
  ?             Toggle this help modal
  q, Ctrl+C     Quit application
`

	modal := helpModalStyle.Render(helpText)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, modal,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("240")),
	)
}

// Run launches the interactive bubbletea TUI.
func Run(report *model.TreeReport, opts Options) error {
	model := NewAppModel(report, opts)
	p := tea.NewProgram(model, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

func nodeMatchesQuery(node *model.TreeNode, query string) bool {
	if query == "" {
		return true
	}
	q := strings.ToLower(query)
	if strings.Contains(strings.ToLower(node.Path), q) || strings.Contains(strings.ToLower(node.Name), q) {
		return true
	}
	for _, child := range node.Children {
		if nodeMatchesQuery(child, query) {
			return true
		}
	}
	return false
}
