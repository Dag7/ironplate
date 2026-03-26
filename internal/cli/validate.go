package cli

import (
	"fmt"
	"strings"

	"github.com/dag7/ironplate/internal/components"
	"github.com/dag7/ironplate/internal/config"
	"github.com/dag7/ironplate/internal/tui"
	"github.com/spf13/cobra"
)

func newValidateCmd() *cobra.Command {
	var strict bool

	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate ironplate.yaml configuration",
		Long:  `Validate the project configuration file for schema correctness, dependency consistency, and port conflicts.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath, err := config.FindConfigFile(".")
			if err != nil {
				return fmt.Errorf("no ironplate project found: %w", err)
			}

			cfg, err := config.Load(configPath)
			if err != nil {
				return fmt.Errorf("schema validation failed: %w", err)
			}

			printer := tui.NewStatusPrinter()
			var errors int
			var warnings int

			fmt.Println()
			fmt.Println(tui.BoldStyle.Render("  Validating " + configPath))

			// 1. Schema validation (already passed if we got here)
			printer.Section("Schema Validation")
			printer.Success("Configuration file parsed and schema validated")

			// 2. Component dependency check
			printer.Section("Component Dependencies")
			if len(cfg.Spec.Infrastructure.Components) > 0 {
				resolved, depErr := components.ResolveDependencies(cfg.Spec.Infrastructure.Components)
				if depErr != nil {
					printer.Error(fmt.Sprintf("Dependency resolution failed: %s", depErr))
					errors++
				} else {
					// Check if resolved list includes components not explicitly listed
					installedSet := make(map[string]bool)
					for _, c := range cfg.Spec.Infrastructure.Components {
						installedSet[c] = true
					}

					missingDeps := false
					for _, r := range resolved {
						if !installedSet[r] {
							printer.Warning(fmt.Sprintf("Component %q is required by dependencies but not listed in config", r))
							warnings++
							missingDeps = true
						}
					}
					if !missingDeps {
						printer.Success("All component dependencies satisfied")
					}

					// Check for unknown components
					for _, c := range cfg.Spec.Infrastructure.Components {
						if components.Get(c) == nil {
							printer.Error(fmt.Sprintf("Unknown component %q", c))
							errors++
						}
					}
				}
			} else {
				printer.Success("No components configured (nothing to check)")
			}

			// 3. Port conflict detection
			printer.Section("Port Conflicts")
			if len(cfg.Spec.Services) > 0 {
				portMap := make(map[int][]string)
				for _, svc := range cfg.Spec.Services {
					if svc.Port > 0 {
						portMap[svc.Port] = append(portMap[svc.Port], svc.Name)
					}
				}

				hasConflict := false
				for port, services := range portMap {
					if len(services) > 1 {
						printer.Error(fmt.Sprintf("Port %d is used by multiple services: %s", port, strings.Join(services, ", ")))
						errors++
						hasConflict = true
					}
				}
				if !hasConflict {
					printer.Success("No port conflicts detected")
				}
			} else {
				printer.Success("No services configured (nothing to check)")
			}

			// 4. Feature consistency
			printer.Section("Feature Consistency")
			{
				installedComponents := make(map[string]bool)
				for _, c := range cfg.Spec.Infrastructure.Components {
					installedComponents[c] = true
				}

				featureComponentMap := map[string]string{
					"hasura":   "hasura",
					"cache":    "redis",
					"dapr":     "dapr",
					"eventbus": "kafka",
				}

				hasIssue := false
				for _, svc := range cfg.Spec.Services {
					for _, feature := range svc.Features {
						requiredComponent, known := featureComponentMap[feature]
						if known && !installedComponents[requiredComponent] {
							printer.Warning(fmt.Sprintf("Service %q uses feature %q but component %q is not installed", svc.Name, feature, requiredComponent))
							warnings++
							hasIssue = true
						}
					}
				}
				if !hasIssue {
					printer.Success("All service features have required components")
				}
			}

			// 5. Language consistency
			printer.Section("Language Consistency")
			{
				typeLanguageMap := map[string]string{
					"node-api":  "node",
					"nextjs":    "node",
					"go-api":    "go",
					"go-worker": "go",
				}

				hasIssue := false
				for _, svc := range cfg.Spec.Services {
					requiredLang, known := typeLanguageMap[svc.Type]
					if known && !cfg.Spec.HasLanguage(requiredLang) {
						printer.Warning(fmt.Sprintf("Service %q has type %q but %q is not in spec.languages", svc.Name, svc.Type, requiredLang))
						warnings++
						hasIssue = true
					}
				}
				if !hasIssue {
					printer.Success("All service types match configured languages")
				}
			}

			// Summary
			fmt.Println()
			if errors > 0 {
				fmt.Printf("  %s Validation failed: %d error(s), %d warning(s)\n", tui.CrossMark, errors, warnings)
				fmt.Println()
				return fmt.Errorf("validation failed with %d error(s)", errors)
			}
			if warnings > 0 && strict {
				fmt.Printf("  %s Validation failed (strict mode): %d warning(s)\n", tui.WarnMark, warnings)
				fmt.Println()
				return fmt.Errorf("validation failed with %d warning(s) in strict mode", warnings)
			}
			if warnings > 0 {
				fmt.Printf("  %s Validation passed with %d warning(s)\n", tui.WarnMark, warnings)
			} else {
				fmt.Printf("  %s Validation passed\n", tui.CheckMark)
			}
			fmt.Println()

			return nil
		},
	}

	cmd.Flags().BoolVar(&strict, "strict", false, "Fail on warnings")

	return cmd
}
