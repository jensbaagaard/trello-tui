package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	primaryColor   = lipgloss.Color("#7C3AED")
	secondaryColor = lipgloss.Color("#06B6D4")
	dimColor       = lipgloss.Color("#6B7280")
	errorColor     = lipgloss.Color("#EF4444")
	cardBorderDim  = lipgloss.Color("#777799")

	// Column styles — simple padding, no background
	columnStyle = lipgloss.NewStyle().
			Padding(0, 1)

	activeColumnStyle = lipgloss.NewStyle().
				Padding(0, 1)

	columnTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(secondaryColor).
				MarginBottom(1)

	// Card styles — rounded border boxes
	cardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(cardBorderDim).
			Padding(0, 1)

	selectedCardStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(primaryColor).
				Bold(true).
				Padding(0, 1)

	// Label pill style — small colored badges
	labelStyle = lipgloss.NewStyle()

	// Status / help
	helpStyle = lipgloss.NewStyle().
			Foreground(dimColor)

	errorStyle = lipgloss.NewStyle().
			Foreground(errorColor)

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor)
)

func labelColor(color string) lipgloss.Style {
	colors := map[string]lipgloss.Color{
		"green":  lipgloss.Color("#10B981"),
		"yellow": lipgloss.Color("#F59E0B"),
		"orange": lipgloss.Color("#F97316"),
		"red":    lipgloss.Color("#EF4444"),
		"purple": lipgloss.Color("#8B5CF6"),
		"blue":   lipgloss.Color("#3B82F6"),
		"sky":    lipgloss.Color("#0EA5E9"),
		"lime":   lipgloss.Color("#84CC16"),
		"pink":   lipgloss.Color("#EC4899"),
		"black":  lipgloss.Color("#374151"),
	}
	c, ok := colors[color]
	if !ok {
		c = lipgloss.Color("#6B7280")
	}
	return labelStyle.Foreground(c).Bold(true)
}
