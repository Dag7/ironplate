package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/ironplate-dev/ironplate/internal/components"
	"github.com/ironplate-dev/ironplate/internal/config"
	"github.com/ironplate-dev/ironplate/internal/engine"
	"github.com/ironplate-dev/ironplate/internal/tui"
	"github.com/ironplate-dev/ironplate/templates"
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

			// 1. Find and load the project config
			cfgPath, err := config.FindConfigFile(".")
			if err != nil {
				return fmt.Errorf("not in an ironplate project: %w", err)
			}

			cfg, err := config.Load(cfgPath)
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			projectRoot := filepath.Dir(cfgPath)

			// 2. Check if the component is already installed
			if cfg.Spec.Infrastructure.HasComponent(componentName) {
				printer.Warning(fmt.Sprintf("Component %q is already installed", componentName))
				return nil
			}

			// 3. Validate the component exists in the registry
			comp := components.Get(componentName)
			if comp == nil {
				available := components.List()
				return fmt.Errorf("unknown component %q — available components: %s", componentName, strings.Join(available, ", "))
			}

			// 4. Resolve dependencies (includes the requested component + its transitive requirements)
			toInstall, err := components.ResolveDependencies([]string{componentName})
			if err != nil {
				return fmt.Errorf("resolve dependencies: %w", err)
			}

			// Filter out components that are already installed
			var newComponents []string
			for _, name := range toInstall {
				if !cfg.Spec.Infrastructure.HasComponent(name) {
					newComponents = append(newComponents, name)
				}
			}

			if len(newComponents) == 0 {
				printer.Warning(fmt.Sprintf("Component %q and all its dependencies are already installed", componentName))
				return nil
			}

			// 5. Print what will be installed
			printer.Section("Adding components")

			if len(newComponents) > 1 {
				var deps []string
				for _, name := range newComponents {
					if name != componentName {
						deps = append(deps, name)
					}
				}
				if len(deps) > 0 {
					printer.Info(fmt.Sprintf("Resolving dependencies: %s requires %s", componentName, strings.Join(deps, ", ")))
				}
			}

			// 6. Check for suggestions
			for _, name := range newComponents {
				c := components.Get(name)
				if c == nil {
					continue
				}
				for _, suggested := range c.Suggests {
					if !cfg.Spec.Infrastructure.HasComponent(suggested) && !contains(newComponents, suggested) {
						printer.Info(fmt.Sprintf("Tip: %q suggests also installing %q — run: iron add %s", name, suggested, suggested))
					}
				}
			}

			// 7. Render templates for each component
			renderer := engine.NewRenderer()
			total := len(newComponents)

			for i, name := range newComponents {
				c := components.Get(name)
				if c == nil {
					continue
				}

				// Add the component to the config before building context
				// so that HasComponent flags are accurate for template rendering
				cfg.Spec.Infrastructure.Components = append(cfg.Spec.Infrastructure.Components, name)

				ctx := engine.NewTemplateContext(cfg)

				printer.Step(i+1, total, fmt.Sprintf("Installing %s — %s", c.Name, c.Description))

				for _, tmplDir := range c.Templates {
					outputPath := filepath.Join(projectRoot, "k8s", "helm", "infra", name)
					if err := renderer.RenderFS(templates.FS, tmplDir, outputPath, ctx); err != nil {
						return fmt.Errorf("render templates for %s: %w", name, err)
					}
				}
				for _, mapping := range c.ExtraTemplates {
					outputPath := filepath.Join(projectRoot, mapping.Output)
					if err := renderer.RenderFS(templates.FS, mapping.Source, outputPath, ctx); err != nil {
						return fmt.Errorf("render extra templates for %s: %w", name, err)
					}
				}
			}

			// 8. Marshal updated config back to ironplate.yaml
			data, err := yaml.Marshal(cfg)
			if err != nil {
				return fmt.Errorf("marshal config: %w", err)
			}

			if err := os.WriteFile(cfgPath, data, 0o644); err != nil {
				return fmt.Errorf("write config: %w", err)
			}

			// 9. Print summary
			fmt.Println()
			printer.Success(fmt.Sprintf("Added %d component(s) to the project", len(newComponents)))
			for _, name := range newComponents {
				c := components.Get(name)
				if c != nil {
					printer.Info(fmt.Sprintf("%s — %s", c.Name, c.Description))
				}
			}
			printer.Info(fmt.Sprintf("Config updated: %s", cfgPath))
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

			// Find and load project config
			cfgPath, err := config.FindConfigFile(".")
			if err != nil {
				return fmt.Errorf("not in an ironplate project: %w", err)
			}

			cfg, err := config.Load(cfgPath)
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			// Check if the component is actually installed
			if !cfg.Spec.Infrastructure.HasComponent(componentName) {
				printer.Warning(fmt.Sprintf("Component %q is not installed", componentName))
				return nil
			}

			// Validate the component exists in the registry (for a better message)
			comp := components.Get(componentName)
			if comp == nil {
				// Still allow removal of unknown components from config
				printer.Warning(fmt.Sprintf("Component %q is not in the registry but will be removed from config", componentName))
			}

			// Check if any other installed component depends on this one
			var dependents []string
			for _, installed := range cfg.Spec.Infrastructure.Components {
				if installed == componentName {
					continue
				}
				c := components.Get(installed)
				if c == nil {
					continue
				}
				for _, req := range c.Requires {
					if req == componentName {
						dependents = append(dependents, installed)
					}
				}
			}

			if len(dependents) > 0 {
				return fmt.Errorf("cannot remove %q — it is required by: %s. Remove those components first",
					componentName, strings.Join(dependents, ", "))
			}

			printer.Section("Removing component")

			// Remove from the components list
			var updated []string
			for _, c := range cfg.Spec.Infrastructure.Components {
				if c != componentName {
					updated = append(updated, c)
				}
			}
			cfg.Spec.Infrastructure.Components = updated

			// Marshal and write updated config
			data, err := yaml.Marshal(cfg)
			if err != nil {
				return fmt.Errorf("marshal config: %w", err)
			}

			if err := os.WriteFile(cfgPath, data, 0o644); err != nil {
				return fmt.Errorf("write config: %w", err)
			}

			// Summary
			fmt.Println()
			if comp != nil {
				printer.Success(fmt.Sprintf("Removed %s (%s) from project configuration", comp.Name, comp.Description))
			} else {
				printer.Success(fmt.Sprintf("Removed %q from project configuration", componentName))
			}
			printer.Warning("Generated files were NOT deleted — review and remove them manually if needed")
			printer.Info(fmt.Sprintf("Config updated: %s", cfgPath))
			fmt.Println()

			return nil
		},
	}

	return cmd
}

// contains checks if a string slice contains a given value.
func contains(slice []string, val string) bool {
	for _, s := range slice {
		if s == val {
			return true
		}
	}
	return false
}
