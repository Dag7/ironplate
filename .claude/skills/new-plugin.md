# Create a New Plugin

Create a new plugin implementing one of the core interfaces: CloudProvider, ServiceGenerator, or InfraComponent.

## Arguments

- `name`: Plugin name (e.g., "aws", "nextjs-app")
- `type`: Plugin type: `cloud-provider`, `service-generator`, `infra-component`

## Steps

### Cloud Provider

1. **Create provider package** at `internal/providers/<name>/`
   - `<name>.go` implementing `plugin.CloudProvider` interface
   - Methods: `Name()`, `Description()`, `GenerateIaC()`, `GenerateCIAuth()`, `RegistryConfig()`, `RequiredAPIs()`, `SupportedComponents()`

2. **Create Pulumi templates** at `templates/iac/pulumi/<name>/`
   - `Pulumi.yaml.tmpl`, `Pulumi.staging.yaml.tmpl`, `Pulumi.production.yaml.tmpl`
   - `src/index.ts.tmpl` with provider-specific infrastructure phases
   - `src/config/types.ts.tmpl` with provider configuration interfaces

3. **Create CI auth templates** at `templates/cicd/<platform>/<name>/`
   - Workflow authentication steps (e.g., Workload Identity for GCP, IRSA for AWS)

4. **Register** in `internal/plugin/loader.go` `RegisterDefaults()`

### Service Generator

1. **Create generator** at `internal/generators/service/<name>.go`
   - Implement `plugin.ServiceGenerator` interface
   - Methods: `Name()`, `Language()`, `ServiceType()`, `Generate()`, `HelmFragment()`, `TiltFragment()`

2. **Create templates** at `templates/service/<name>/`
   - Source code templates, config templates
   - `helm/` subdirectory with Helm chart templates

3. **Register** in `internal/plugin/loader.go`

### Infrastructure Component

1. **Create component** definition in `internal/components/registry.go`
   - Add to `builtinComponents` with proper tier, dependencies, templates

2. **Create templates** at `templates/components/<name>/`
   - `helm/` with Helm chart templates
   - `tiltfile-entry.tmpl` with Tilt setup function

3. **Add CLAUDE.md and skill templates** if applicable

## Checklist

- [ ] Plugin implements the full interface contract
- [ ] Templates created and render without errors
- [ ] Plugin registered in `RegisterDefaults()`
- [ ] Unit tests for plugin logic
- [ ] Golden file test for template output
- [ ] `iron list` shows the new plugin
