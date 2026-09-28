package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	colorPrimary   = lipgloss.Color("39")  // Blue / Cyan
	colorSecondary = lipgloss.Color("245") // Dim Gray
	colorGreen     = lipgloss.Color("42")  // Green
	colorRed       = lipgloss.Color("196") // Red
	colorHighlight = lipgloss.Color("212") // Pink / Magenta
	colorSelected  = lipgloss.Color("237") // Dark Gray background

	// Header Styles
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("15")).
			Background(colorPrimary).
			Padding(0, 1)

	headerInfoStyle = lipgloss.NewStyle().
			Foreground(colorSecondary)

	// Tree Pane Styles
	paneStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1)

	focusedPaneStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorPrimary).
				Padding(0, 1)

	paneTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPrimary)

	// Tree Row Styles
	cursorStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("15")).
			Background(colorSelected)

	dirStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("75"))

	fileStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	addedStyle = lipgloss.NewStyle().
			Foreground(colorGreen)

	deletedStyle = lipgloss.NewStyle().
			Foreground(colorRed)

	// Detail Pane Styles
	detailHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("15"))

	// Status / Footer Bar Styles
	statusBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("250")).
			Background(lipgloss.Color("236")).
			Padding(0, 1)

	helpKeyStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPrimary)

	helpDescStyle = lipgloss.NewStyle().
			Foreground(colorSecondary)

	// Overlay / Help Modal Styles
	helpModalStyle = lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(colorPrimary).
			Background(lipgloss.Color("235")).
			Padding(1, 2)
)
