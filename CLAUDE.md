# Claude Code Rules for Ironplate

## Language & Tooling

- **Go 1.24+** with Go modules
- Use `go fmt`, `go vet`, `golangci-lint`
- Do not use CGO unless absolutely necessary
- CLI framework: **Cobra** (`github.com/spf13/cobra`)
- TUI: **Huh** (`github.com/charmbracelet/huh`) for forms, **Lipgloss** for styling
- YAML: `gopkg.in/yaml.v3`
- Templates: Go `text/template` with custom function map + sprig
- Testing: `github.com/stretchr/testify`

## Project Structure

```
cmd/iron/              CLI entry point (main.go)
internal/
  cli/                 Command implementations (cobra commands)
  config/              Configuration loading and validation (ironplate.yaml)
  engine/              Template rendering engine
  plugin/              Plugin system interfaces and registry
  providers/           Cloud provider implementations (GCP, AWS stubs)
  generators/          Service, package, and claude generators
  components/          Infrastructure component registry
  scaffold/            Project scaffolding orchestrator
  tui/                 Terminal UI components
  version/             Build-time version info
pkg/
  fsutil/              File system utilities
  executil/            Shell command execution helpers
templates/             Embedded template files (//go:embed)
testdata/              Test fixtures and golden files
```

## SOLID Principles (Mandatory)

- **Single Responsibility**: Each package has one concern. `config/` parses config, `engine/` renders templates, `plugin/` manages plugins.
- **Open/Closed**: Plugin system allows extension without modifying core. New cloud providers, service types, and infra components are added as implementations, never by modifying existing code.
- **Liskov Substitution**: All cloud providers implement `CloudProvider`, all generators implement `ServiceGenerator`. They are fully interchangeable.
- **Interface Segregation**: Small interfaces: `CloudProvider`, `ServiceGenerator`, `InfraComponent`, `PackageGenerator` -- not one god `Plugin` interface.
- **Dependency Inversion**: Accept interfaces in constructors, return concrete structs. Plugin registry depends on interfaces, not implementations.

## Design Patterns Used

| Pattern | Where | Purpose |
|---------|-------|---------|
| **Strategy** | `internal/plugin/types.go` | Interchangeable cloud providers (GCP/AWS/Azure) |
| **Registry** | `internal/plugin/registry.go`, `internal/components/registry.go` | Register and discover plugins and components |
| **Template Method** | `internal/scaffold/project.go` | Shared scaffolding flow with variable steps |
| **Factory** | `internal/config/defaults.go` | Create default configurations |
| **Builder** | `internal/engine/context.go` | Build template contexts from configuration |

## Template Development

- Templates are embedded using `//go:embed` in the `templates/` directory
- Files ending in `.tmpl` are rendered through Go's `text/template` engine
- Non-`.tmpl` files are copied verbatim
- Custom delimiters `[[` / `]]` for files containing `{{ }}` (Helm templates, GitHub Actions)
- Always use the custom function map from `internal/engine/funcmap.go`
- Template variables: `{{ .Project.Metadata.Name }}`, `{{ .Computed.HasNode }}`, etc.

## Error Handling

- Return errors, never panic (except in `main`)
- Wrap errors with context: `fmt.Errorf("failed to generate %s: %w", name, err)`
- User-facing errors must be human-readable (no stack traces in output)

## Testing

- **Table-driven tests** for all pure functions
- **Golden file tests** in `testdata/golden/` for template output
- **Integration tests** with `//go:build integration` tag
- Use `testify/assert` and `testify/require`
- Run: `make test` (unit), `make test-integration` (e2e)

## Reference

- The doorz repository at `~/Desktop/repo/doorz` is the reference for all templates
- Key source files: `.devcontainer/`, `dockerfiles/`, `utils/tilt/`, `k8s/helm/`, `iac/pulumi/`, `.github/`, `CLAUDE.md`

## Skills

| Skill | Description |
|-------|-------------|
| `/new-template` | Add a new template category |
| `/new-plugin` | Create a new plugin (cloud provider, service type) |
| `/new-generator` | Add a new generator command |
| `/add-component-template` | Add templates for a new infrastructure component |
| `/new-test` | Generate test scaffolding |

## Restrictions

- Do not modify generated template output manually -- edit templates instead
- Do not add CGO dependencies
- Do not use global mutable state -- pass dependencies through constructors
- Do not create markdown files unless explicitly asked
