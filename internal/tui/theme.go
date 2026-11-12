// Package tui provides the interactive wizard and shared terminal styling.
package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	accentColor  = lipgloss.AdaptiveColor{Light: "#6D28D9", Dark: "#A78BFA"}
	mutedColor   = lipgloss.AdaptiveColor{Light: "#6B7280", Dark: "#9CA3AF"}
	successColor = lipgloss.AdaptiveColor{Light: "#047857", Dark: "#34D399"}
)

var (
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(accentColor)
	stepStyle  = lipgloss.NewStyle().Foreground(successColor).Bold(true)
	mutedStyle = lipgloss.NewStyle().Foreground(mutedColor)
	labelStyle = lipgloss.NewStyle().Bold(true)
	boxStyle   = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accentColor).
			Padding(0, 2).
			MarginTop(1).
			MarginBottom(1)
)

// Title renders an accent-colored heading.
func Title(text string) string {
	return titleStyle.Render(text)
}

// Step renders a completed generation stage.
func Step(text string) string {
	return stepStyle.Render("✓") + " " + text
}

// Muted renders secondary text.
func Muted(text string) string {
	return mutedStyle.Render(text)
}

// Box renders a titled summary panel.
func Box(title string, lines []string) string {
	content := make([]string, 0, len(lines)+2)
	content = append(content, titleStyle.Render(title))

	if len(lines) > 0 {
		content = append(content, "")
	}

	content = append(content, lines...)

	return boxStyle.Render(strings.Join(content, "\n"))
}

// Field renders a labeled summary value.
func Field(label, value string) string {
	return labelStyle.Render(label+":") + " " + value
}
