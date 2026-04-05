package cli

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/dag7/ironplate/internal/tiltmgr"
	"github.com/dag7/ironplate/internal/tui"
)

func printTiltStatus() error {
	printer := tui.NewStatusPrinter()

	if !tiltmgr.IsRunning() {
		printer.Info("Tilt is not running.")
		return nil
	}

	resources, err := tiltmgr.GetStatus()
	if err != nil {
		return err
	}

	if len(resources) == 0 {
		printer.Info("No resources found.")
		return nil
	}

	// Summary counts
	summary, _ := tiltmgr.GetStatusSummary()
	if summary != nil {
		fmt.Println()
		parts := []string{
			fmt.Sprintf("Total: %d", summary.Total),
		}
		if summary.Running > 0 {
			parts = append(parts, tui.SuccessStyle.Render(fmt.Sprintf("● %d running", summary.Running)))
		}
		if summary.Pending > 0 {
			parts = append(parts, tui.WarningStyle.Render(fmt.Sprintf("◐ %d pending", summary.Pending)))
		}
		if summary.Error > 0 {
			parts = append(parts, tui.ErrorStyle.Render(fmt.Sprintf("✗ %d error", summary.Error)))
		}
		if summary.Disabled > 0 {
			parts = append(parts, tui.MutedStyle.Render(fmt.Sprintf("◌ %d disabled", summary.Disabled)))
		}
		fmt.Printf("  %s\n", strings.Join(parts, "  "))
	}

	// Resource table
	fmt.Printf("\n  %-30s %-12s %-15s %s\n", "NAME", "STATUS", "TYPE", "UPDATE")
	fmt.Printf("  %s\n", strings.Repeat("─", 70))

	for _, r := range resources {
		icon, statusStyle := tiltStatusStyle(r.Status)
		fmt.Printf("  %s %-28s %s  %-15s %s\n",
			statusStyle.Render(icon),
			r.Name,
			statusStyle.Render(fmt.Sprintf("%-10s", r.Status)),
			r.Type,
			r.UpdateStatus,
		)
	}
	fmt.Println()
	return nil
}

func tiltStatusStyle(status string) (string, lipgloss.Style) {
	switch status {
	case "ok":
		return "●", tui.SuccessStyle
	case "pending":
		return "◐", tui.WarningStyle
	case "error":
		return "✗", tui.ErrorStyle
	case "disabled":
		return "◌", tui.MutedStyle
	default:
		return "○", tui.MutedStyle
	}
}

func printProfileList(pm *tiltmgr.ProfileManager) error {
	profiles, err := pm.List()
	if err != nil {
		return err
	}

	if len(profiles) == 0 {
		tui.NewStatusPrinter().Info("No profiles found. Create one with: iron tilt profile create <name>")
		return nil
	}

	fmt.Printf("\n  %-20s %-10s %-10s %s\n", "NAME", "SERVICES", "INFRA", "DESCRIPTION")
	fmt.Printf("  %s\n", strings.Repeat("─", 60))

	for _, p := range profiles {
		fmt.Printf("  %-20s %-10d %-10d %s\n",
			p.Name, len(p.Services), len(p.Infra), p.Description)
	}
	fmt.Printf("\n  %s\n\n", tui.MutedStyle.Render(fmt.Sprintf("Total: %d profiles", len(profiles))))
	return nil
}

func printProfileBox(profile *tiltmgr.Profile) {
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(tui.ColorPrimary).
		Padding(1, 2).
		Width(50)

	var content strings.Builder
	content.WriteString(tui.BoldStyle.Render(profile.Name) + "\n")
	if profile.Description != "" {
		content.WriteString(tui.MutedStyle.Render(profile.Description) + "\n")
	}
	content.WriteString("\n")

	if len(profile.Infra) > 0 {
		content.WriteString(tui.BoldStyle.Render("Infrastructure") + "\n")
		for _, r := range profile.Infra {
			content.WriteString("  " + r + "\n")
		}
		content.WriteString("\n")
	}
	if len(profile.Services) > 0 {
		content.WriteString(tui.BoldStyle.Render("Services") + "\n")
		for _, r := range profile.Services {
			content.WriteString("  " + r + "\n")
		}
	}

	total := len(profile.Infra) + len(profile.Services)
	content.WriteString("\n" + tui.MutedStyle.Render(fmt.Sprintf("Total: %d resources", total)))

	fmt.Println()
	fmt.Println(boxStyle.Render(content.String()))
}

func printServiceList(discovered *tiltmgr.DiscoveredResources) error {
	fmt.Printf("\n  %-30s %-15s %s\n", "NAME", "GROUP", "TYPE")
	fmt.Printf("  %s\n", strings.Repeat("─", 55))

	for _, name := range discovered.Infra {
		fmt.Printf("  %-30s %-15s %s\n", name, "infra", tui.MutedStyle.Render("infra"))
	}
	for _, svc := range discovered.Services {
		fmt.Printf("  %-30s %-15s %s\n", svc.Name, svc.Group, "service")
	}

	fmt.Printf("\n  %s\n\n",
		tui.MutedStyle.Render(fmt.Sprintf("%d services, %d infra resources",
			len(discovered.Services), len(discovered.Infra))))
	return nil
}

func printServiceGroups(discovered *tiltmgr.DiscoveredResources) error {
	groups := make(map[string][]string)
	if len(discovered.Infra) > 0 {
		groups["infra"] = discovered.Infra
	}
	for _, svc := range discovered.Services {
		groups[svc.Group] = append(groups[svc.Group], svc.Name)
	}

	fmt.Println()
	for group, members := range groups {
		fmt.Printf("  %s %s\n",
			tui.BoldStyle.Render(group),
			tui.MutedStyle.Render(fmt.Sprintf("(%d)", len(members))))
		for _, m := range members {
			fmt.Printf("    • %s\n", m)
		}
		fmt.Println()
	}
	return nil
}
