package cli

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/dag7/ironplate/internal/components"
	"github.com/dag7/ironplate/internal/engine"
	"github.com/dag7/ironplate/internal/tui"
	"github.com/dag7/ironplate/templates"
)

func newAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add <component>",
		Short: "Add a component to an existing project",
		Long: `Add infrastructure components, cloud providers, skills, or environments
to an existing ironplate project.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			componentName := args[0]
			printer := tui.NewStatusPrinter()

			pc, err := loadProject()
			if err != nil {
				return err
			}
			cfg := pc.Config

			if cfg.Spec.Infrastructure.HasComponent(componentName) {
				printer.Warning(fmt.Sprintf("Component %q is already installed", componentName))
				return nil
			}

			comp := components.Get(componentName)
			if comp == nil {
				return fmt.Errorf("unknown component %q — available: %s", componentName, strings.Join(components.List(), ", "))
			}

			toInstall, err := components.ResolveDependencies([]string{componentName})
			if err != nil {
				return fmt.Errorf("resolve dependencies: %w", err)
			}

			newComponents := filterNew(toInstall, cfg.Spec.Infrastructure.Components)
			if len(newComponents) == 0 {
				printer.Warning(fmt.Sprintf("Component %q and all its dependencies are already installed", componentName))
				return nil
			}

			printer.Section("Adding components")
			printDependencyInfo(printer, componentName, newComponents)
			printSuggestions(printer, newComponents, cfg.Spec.Infrastructure.Components)

			// Render templates — add all components to config first so flags are accurate
			cfg.Spec.Infrastructure.Components = append(cfg.Spec.Infrastructure.Components, newComponents...)
			ctx := engine.NewTemplateContext(cfg)
			renderer := engine.NewRenderer()

			for i, name := range newComponents {
				c := components.Get(name)
				if c == nil {
					continue
				}

				printer.Step(i+1, len(newComponents), fmt.Sprintf("Installing %s — %s", c.Name, c.Description))

				if err := renderComponent(renderer, pc.ProjectRoot, name, c, ctx); err != nil {
					return err
				}
			}

			if err := pc.saveConfig(); err != nil {
				return fmt.Errorf("save config: %w", err)
			}

			fmt.Println()
			printer.Success(fmt.Sprintf("Added %d component(s) to the project", len(newComponents)))
			for _, name := range newComponents {
				if c := components.Get(name); c != nil {
					printer.Info(fmt.Sprintf("%s — %s", c.Name, c.Description))
				}
			}
			fmt.Println()

			return nil
		},
	}

	return cmd
}

func newRemoveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove <component>",
		Short: "Remove a component from the project configuration",
		Long: `Remove an infrastructure component from the ironplate project configuration.

Note: This removes the component from ironplate.yaml but does NOT delete
any previously generated files. You may want to clean those up manually.`,
		Aliases: []string{"rm"},
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			componentName := args[0]
			printer := tui.NewStatusPrinter()

			pc, err := loadProject()
			if err != nil {
				return err
			}
			cfg := pc.Config

			if !cfg.Spec.Infrastructure.HasComponent(componentName) {
				printer.Warning(fmt.Sprintf("Component %q is not installed", componentName))
				return nil
			}

			comp := components.Get(componentName)
			if comp == nil {
				printer.Warning(fmt.Sprintf("Component %q is not in the registry but will be removed from config", componentName))
			}

			if dependents := findDependents(componentName, cfg.Spec.Infrastructure.Components); len(dependents) > 0 {
				return fmt.Errorf("cannot remove %q — required by: %s. Remove those first",
					componentName, strings.Join(dependents, ", "))
			}

			printer.Section("Removing component")

			cfg.Spec.Infrastructure.Components = removeFromSlice(cfg.Spec.Infrastructure.Components, componentName)

			if err := pc.saveConfig(); err != nil {
				return fmt.Errorf("save config: %w", err)
			}

			fmt.Println()
			if comp != nil {
				printer.Success(fmt.Sprintf("Removed %s (%s) from project configuration", comp.Name, comp.Description))
			} else {
				printer.Success(fmt.Sprintf("Removed %q from project configuration", componentName))
			}
			printer.Warning("Generated files were NOT deleted — review and remove them manually if needed")
			fmt.Println()

			return nil
		},
	}

	return cmd
}

// --- helpers for add/remove ---

// filterNew returns items from candidates that are not in existing.
func filterNew(candidates, existing []string) []string {
	installed := make(map[string]bool, len(existing))
	for _, c := range existing {
		installed[c] = true
	}
	var result []string
	for _, name := range candidates {
		if !installed[name] {
			result = append(result, name)
		}
	}
	return result
}

// findDependents returns installed components that depend on the given component.
func findDependents(componentName string, installed []string) []string {
	var dependents []string
	for _, name := range installed {
		if name == componentName {
			continue
		}
		c := components.Get(name)
		if c == nil {
			continue
		}
		for _, req := range c.Requires {
			if req == componentName {
				dependents = append(dependents, name)
				break
			}
		}
	}
	return dependents
}

// removeFromSlice returns a new slice without the given value.
func removeFromSlice(slice []string, val string) []string {
	result := make([]string, 0, len(slice))
	for _, s := range slice {
		if s != val {
			result = append(result, s)
		}
	}
	return result
}

// renderComponent renders all templates for an infrastructure component.
func renderComponent(renderer *engine.Renderer, projectRoot, name string, comp *components.Component, ctx *engine.TemplateContext) error {
	for _, tmplDir := range comp.Templates {
		outputPath := filepath.Join(projectRoot, "k8s", "helm", "infra", name)
		if err := renderer.RenderFS(templates.FS, tmplDir, outputPath, ctx); err != nil {
			return fmt.Errorf("render templates for %s: %w", name, err)
		}
	}
	for _, mapping := range comp.ExtraTemplates {
		outputPath := filepath.Join(projectRoot, mapping.Output)
		if err := renderer.RenderFS(templates.FS, mapping.Source, outputPath, ctx); err != nil {
			return fmt.Errorf("render extra templates for %s: %w", name, err)
		}
	}
	return nil
}

// printDependencyInfo prints which dependencies will be auto-installed.
func printDependencyInfo(printer *tui.StatusPrinter, requested string, newComponents []string) {
	if len(newComponents) <= 1 {
		return
	}
	var deps []string
	for _, name := range newComponents {
		if name != requested {
			deps = append(deps, name)
		}
	}
	if len(deps) > 0 {
		printer.Info(fmt.Sprintf("Resolving dependencies: %s requires %s", requested, strings.Join(deps, ", ")))
	}
}

// printSuggestions prints component install suggestions.
func printSuggestions(printer *tui.StatusPrinter, newComponents, existing []string) {
	newSet := make(map[string]bool, len(newComponents))
	for _, n := range newComponents {
		newSet[n] = true
	}
	existingSet := make(map[string]bool, len(existing))
	for _, e := range existing {
		existingSet[e] = true
	}

	for _, name := range newComponents {
		c := components.Get(name)
		if c == nil {
			continue
		}
		for _, suggested := range c.Suggests {
			if !existingSet[suggested] && !newSet[suggested] {
				printer.Info(fmt.Sprintf("Tip: %q suggests also installing %q — run: iron add %s", name, suggested, suggested))
			}
		}
	}
}

