package ui

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	// Colors
	PrimaryColor = lipgloss.Color("#7C3AED") // Purple
	SuccessColor = lipgloss.Color("#10B981") // Green
	ErrorColor   = lipgloss.Color("#EF4444") // Red
	WarningColor = lipgloss.Color("#F59E0B") // Orange
	InfoColor    = lipgloss.Color("#3B82F6") // Blue
	MutedColor   = lipgloss.Color("#6B7280") // Gray

	// Base styles
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(PrimaryColor).
			MarginTop(1).
			MarginBottom(1)

	SubtitleStyle = lipgloss.NewStyle().
			Foreground(MutedColor).
			Italic(true)

	SuccessStyle = lipgloss.NewStyle().
			Foreground(SuccessColor).
			Bold(true)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(ErrorColor).
			Bold(true)

	WarningStyle = lipgloss.NewStyle().
			Foreground(WarningColor).
			Bold(true)

	InfoStyle = lipgloss.NewStyle().
			Foreground(InfoColor)

	// Box styles
	BoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(PrimaryColor).
			Padding(1, 2)

	SuccessBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(SuccessColor).
			Padding(1, 2)

	ErrorBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ErrorColor).
			Padding(1, 2)

	// Banner style
	BannerStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(PrimaryColor).
			Padding(0, 2).
			MarginBottom(1).
			Width(42).
			Align(lipgloss.Center)
)

// Logo returns the OpenCore ASCII logo
func Logo() string {
	logo := `
  ◆ OpenCore CLI
  By Newcore Network
`
	return BannerStyle.Render(logo)
}

// Success formats a success message
func Success(msg string) string {
	return SuccessStyle.Render("✓ ") + msg
}

// Error formats an error message
func Error(msg string) string {
	return ErrorStyle.Render("✗ ") + msg
}

// Warning formats a warning message
func Warning(msg string) string {
	return WarningStyle.Render("⚠ ") + msg
}

// Info formats an info message
func Info(msg string) string {
	return InfoStyle.Render("ℹ ") + msg
}

// Muted formats a muted message
func Muted(msg string) string {
	return lipgloss.NewStyle().Foreground(MutedColor).Render(msg)
}
