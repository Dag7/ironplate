package cli

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/dag7/ironplate/internal/config"
	"github.com/dag7/ironplate/internal/manifest"
	"github.com/dag7/ironplate/internal/scaffold"
	"github.com/dag7/ironplate/internal/tui"
	ironversion "github.com/dag7/ironplate/internal/version"
	"github.com/dag7/ironplate/templates"
	"github.com/spf13/cobra"
)

func newUpdateCmd() *cobra.Command {
	var (
		force  bool
		dryRun bool
	)

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update project files to match the latest ironplate templates",
		Long: `Re-renders all templates using your ironplate.yaml config and applies changes
to files that haven't been manually modified. Files you've edited are flagged
as conflicts and shown with a diff for manual review.

Protected paths (apps/, packages/, ironplate.yaml) are never touched.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUpdate(force, dryRun)
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Apply all changes without prompting (overwrite conflicts)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would change without applying")

	return cmd
}

func runUpdate(force, dryRun bool) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	// Load project config
	cfgPath, err := config.FindConfigFile(cwd)
	if err != nil {
		return fmt.Errorf("not in an ironplate project: %w", err)
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return err
	}

	projectDir := strings.TrimSuffix(cfgPath, "/ironplate.yaml")

	// Load existing manifest (may be nil for older projects)
	m, err := manifest.Load(projectDir)
	if err != nil {
		return fmt.Errorf("load manifest: %w", err)
	}

	printer := tui.NewStatusPrinter()

	if m == nil {
		printer.Warning("No manifest found — this project was generated before update tracking.")
		printer.Info("All changed files will be treated as conflicts (requiring confirmation).")
		fmt.Println()
	} else {
		printer.Info(fmt.Sprintf("Project generated with iron %s, updating to %s",
			m.Version, ironversion.Short()))
		fmt.Println()
	}

	// Compute changes
	printer.Info("Re-rendering templates...")
	updater := scaffold.NewUpdater(cfg, projectDir, templates.FS, m)
	changes, err := updater.ComputeChanges()
	if err != nil {
		return fmt.Errorf("compute changes: %w", err)
	}

	if len(changes) == 0 {
		printer.Success("Project is up to date — no changes needed.")
		return nil
	}

	// Sort changes by path for deterministic output
	sort.Slice(changes, func(i, j int) bool {
		return changes[i].RelPath < changes[j].RelPath
	})

	// Classify changes
	var updates, conflicts, newFiles, removed []scaffold.FileChange
	for _, c := range changes {
		switch c.ChangeType {
		case scaffold.ChangeUpdate:
			updates = append(updates, c)
		case scaffold.ChangeConflict:
			conflicts = append(conflicts, c)
		case scaffold.ChangeNew:
			newFiles = append(newFiles, c)
		case scaffold.ChangeRemoved:
			removed = append(removed, c)
		}
	}

	// Print summary
	printUpdateSummary(updates, conflicts, newFiles, removed)

	if dryRun {
		fmt.Println()
		printChangeDiffs(changes)
		printer.Info("Dry run — no files were modified.")
		return nil
	}

	// Apply safe updates automatically
	applied := 0
	if len(updates) > 0 {
		fmt.Println()
		printer.Info(fmt.Sprintf("Applying %d safe updates...", len(updates)))
		for _, c := range updates {
			if err := scaffold.ApplyChange(projectDir, c); err != nil {
				printer.Error(fmt.Sprintf("  Failed: %s: %s", c.RelPath, err))
			} else {
				fmt.Printf("  %s %s\n", tui.CheckMark, c.RelPath)
				applied++
			}
		}
	}

	// Apply new files automatically
	if len(newFiles) > 0 {
		fmt.Println()
		printer.Info(fmt.Sprintf("Creating %d new files...", len(newFiles)))
		for _, c := range newFiles {
			if err := scaffold.ApplyChange(projectDir, c); err != nil {
				printer.Error(fmt.Sprintf("  Failed: %s: %s", c.RelPath, err))
			} else {
				fmt.Printf("  %s %s\n", tui.CheckMark, c.RelPath)
				applied++
			}
		}
	}

	// Handle conflicts interactively (or force)
	if len(conflicts) > 0 {
		fmt.Println()
		if force {
			printer.Warning(fmt.Sprintf("Force-overwriting %d conflicts...", len(conflicts)))
			for _, c := range conflicts {
				if err := scaffold.ApplyChange(projectDir, c); err != nil {
					printer.Error(fmt.Sprintf("  Failed: %s: %s", c.RelPath, err))
				} else {
					fmt.Printf("  %s %s (overwritten)\n", tui.WarnMark, c.RelPath)
					applied++
				}
			}
		} else {
			printer.Warning(fmt.Sprintf("%d file(s) have local modifications:", len(conflicts)))
			for _, c := range conflicts {
				accepted, err := promptConflict(c)
				if err != nil {
					return err
				}
				if accepted {
					if err := scaffold.ApplyChange(projectDir, c); err != nil {
						printer.Error(fmt.Sprintf("  Failed: %s: %s", c.RelPath, err))
					} else {
						fmt.Printf("  %s %s (overwritten)\n", tui.CheckMark, c.RelPath)
						applied++
					}
				} else {
					fmt.Printf("  %s %s (skipped)\n", tui.MutedStyle.Render("○"), c.RelPath)
				}
			}
		}
	}

	// Handle removed files
	if len(removed) > 0 {
		fmt.Println()
		printer.Warning("Files no longer generated by templates (kept):")
		for _, c := range removed {
			fmt.Printf("  %s %s\n", tui.MutedStyle.Render("−"), c.RelPath)
		}
	}

	// Rebuild manifest from current project state
	scaffolder := scaffold.NewScaffolder(cfg, projectDir, templates.FS)
	if err := scaffolder.WriteManifestOnly(); err != nil {
		printer.Warning("Could not update manifest: " + err.Error())
	}

	fmt.Println()
	printer.Success(fmt.Sprintf("Update complete: %d file(s) updated.", applied))
	return nil
}

func printUpdateSummary(updates, conflicts, newFiles, removed []scaffold.FileChange) {
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(tui.ColorSecondary).
		Padding(0, 2).
		Width(50)

	var content strings.Builder
	content.WriteString(tui.BoldStyle.Render("Update Summary") + "\n\n")
	if len(updates) > 0 {
		fmt.Fprintf(&content, "  %s %d file(s) to update\n", tui.CheckMark, len(updates))
	}
	if len(newFiles) > 0 {
		fmt.Fprintf(&content, "  %s %d new file(s)\n", tui.SuccessStyle.Render("+"), len(newFiles))
	}
	if len(conflicts) > 0 {
		fmt.Fprintf(&content, "  %s %d conflict(s) (local modifications)\n", tui.WarnMark, len(conflicts))
	}
	if len(removed) > 0 {
		fmt.Fprintf(&content, "  %s %d removed from templates\n", tui.MutedStyle.Render("−"), len(removed))
	}

	fmt.Println()
	fmt.Println(boxStyle.Render(content.String()))
}

func printChangeDiffs(changes []scaffold.FileChange) {
	for _, c := range changes {
		printFileDiff(c)
	}
}

func promptConflict(c scaffold.FileChange) (bool, error) {
	fmt.Println()
	printFileDiff(c)

	var accept bool
	err := huh.NewConfirm().
		Title(fmt.Sprintf("Overwrite %s?", c.RelPath)).
		Description("This file has local modifications that will be lost.").
		Value(&accept).
		Run()
	if err != nil {
		return false, err
	}
	return accept, nil
}

func printFileDiff(c scaffold.FileChange) {
	headerStyle := lipgloss.NewStyle().Bold(true)
	addStyle := lipgloss.NewStyle().Foreground(tui.ColorSuccess)
	delStyle := lipgloss.NewStyle().Foreground(tui.ColorError)

	var label string
	switch c.ChangeType {
	case scaffold.ChangeUpdate:
		label = tui.SuccessStyle.Render("[update]")
	case scaffold.ChangeConflict:
		label = tui.WarningStyle.Render("[conflict]")
	case scaffold.ChangeNew:
		label = tui.SuccessStyle.Render("[new]")
	case scaffold.ChangeRemoved:
		label = tui.MutedStyle.Render("[removed]")
	}

	fmt.Printf("%s %s\n", label, headerStyle.Render(c.RelPath))

	if c.OldContent == nil && c.NewContent != nil {
		// New file — show first 20 lines
		lines := strings.SplitAfter(string(c.NewContent), "\n")
		limit := 20
		if len(lines) < limit {
			limit = len(lines)
		}
		for _, line := range lines[:limit] {
			fmt.Print(addStyle.Render("+ " + strings.TrimRight(line, "\n") + "\n"))
		}
		if len(lines) > 20 {
			fmt.Printf(tui.MutedStyle.Render("  ... and %d more lines\n"), len(lines)-20)
		}
		return
	}

	if c.NewContent == nil {
		// Removed — show first 5 lines
		lines := strings.SplitAfter(string(c.OldContent), "\n")
		limit := 5
		if len(lines) < limit {
			limit = len(lines)
		}
		for _, line := range lines[:limit] {
			fmt.Print(delStyle.Render("- " + strings.TrimRight(line, "\n") + "\n"))
		}
		if len(lines) > 5 {
			fmt.Printf(tui.MutedStyle.Render("  ... and %d more lines\n"), len(lines)-5)
		}
		return
	}

	// Both exist — show a simple line-by-line diff
	oldLines := strings.Split(string(c.OldContent), "\n")
	newLines := strings.Split(string(c.NewContent), "\n")

	diff := simpleDiff(oldLines, newLines)
	shown := 0
	for _, d := range diff {
		if shown >= 40 {
			remaining := len(diff) - shown
			if remaining > 0 {
				fmt.Printf(tui.MutedStyle.Render("  ... and %d more diff lines\n"), remaining)
			}
			break
		}
		switch d.op {
		case diffAdd:
			fmt.Print(addStyle.Render("+ " + d.line + "\n"))
			shown++
		case diffDel:
			fmt.Print(delStyle.Render("- " + d.line + "\n"))
			shown++
		}
	}
}

// Simple line-based diff (no LCS — just show removed/added lines in changed hunks).
type diffOp int

const (
	diffKeep diffOp = iota
	diffAdd
	diffDel
)

type diffLine struct {
	op   diffOp
	line string
}

func simpleDiff(oldLines, newLines []string) []diffLine {
	// Build a set of old and new lines to detect additions and removals.
	// For a proper diff we'd use Myers or patience, but for an update preview
	// this approach is sufficient — we use the LCS-free approach:
	// walk both sequences, output context-aware chunks.
	oldSet := make(map[string]int, len(oldLines))
	for _, l := range oldLines {
		oldSet[l]++
	}
	newSet := make(map[string]int, len(newLines))
	for _, l := range newLines {
		newSet[l]++
	}

	var result []diffLine

	// Show lines removed (in old but not in new)
	for _, l := range oldLines {
		if newSet[l] > 0 {
			newSet[l]--
			continue
		}
		result = append(result, diffLine{diffDel, l})
	}

	// Reset newSet
	newSet = make(map[string]int, len(newLines))
	for _, l := range newLines {
		newSet[l]++
	}
	oldSet2 := make(map[string]int, len(oldLines))
	for _, l := range oldLines {
		oldSet2[l]++
	}

	// Show lines added (in new but not in old)
	for _, l := range newLines {
		if oldSet2[l] > 0 {
			oldSet2[l]--
			continue
		}
		result = append(result, diffLine{diffAdd, l})
	}

	return result
}
