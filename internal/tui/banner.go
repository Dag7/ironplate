package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/ironplate-dev/ironplate/internal/version"
)

const banner = `
 _                       _       _
(_)_ __ ___  _ __  _ __ | | __ _| |_ ___
| | '__/ _ \| '_ \| '_ \| |/ _` + "`" + ` | __/ _ \
| | | | (_) | | | | |_) | | (_| | ||  __/
|_|_|  \___/|_| |_| .__/|_|\__,_|\__\___|
                   |_|                     `

// PrintBanner displays the ironplate ASCII banner.
func PrintBanner() {
	style := lipgloss.NewStyle().Foreground(ColorPrimary)
	fmt.Println(style.Render(banner))

	versionLine := MutedStyle.Render(fmt.Sprintf("  %s", version.Short()))
	fmt.Println(versionLine)
	fmt.Println()
}
