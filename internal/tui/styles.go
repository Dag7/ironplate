// Package tui provides terminal UI components for the iron CLI.
package tui

import (
	"github.com/charmbracelet/lipgloss"
)

// Color palette for consistent styling.
var (
	ColorPrimary   = lipgloss.Color("#FF6B35") // Iron orange
	ColorSecondary = lipgloss.Color("#4ECDC4") // Teal
	ColorSuccess   = lipgloss.Color("#2ECC71") // Green
	ColorWarning   = lipgloss.Color("#F1C40F") // Yellow
	ColorError     = lipgloss.Color("#E74C3C") // Red
	ColorMuted     = lipgloss.Color("#95A5A6") // Gray
)

// Reusable styles.
var (
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary).
			MarginBottom(1)

	SuccessStyle = lipgloss.NewStyle().
			Foreground(ColorSuccess)

	WarningStyle = lipgloss.NewStyle().
			Foreground(ColorWarning)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(ColorError).
			Bold(true)

	MutedStyle = lipgloss.NewStyle().
			Foreground(ColorMuted)

	BoldStyle = lipgloss.NewStyle().
			Bold(true)

	CheckMark = SuccessStyle.Render("✓")
	CrossMark = ErrorStyle.Render("✗")
	WarnMark  = WarningStyle.Render("!")
	ArrowMark = lipgloss.NewStyle().Foreground(ColorPrimary).Render("→")
)
