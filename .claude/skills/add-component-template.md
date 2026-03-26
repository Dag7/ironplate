# Add Templates for a New Infrastructure Component

Add Helm charts, Tilt config, and skills for a new infrastructure component.

## Arguments

- `component`: Component name (e.g., "rabbitmq", "mongodb", "vault")

## Steps

1. **Define component** in `internal/components/registry.go`
   ```go
   "<component>": {
       Name:        "<component>",
       Description: "Description of the component",
       Tier:        <tier>,           // 0=early, 1=mid, 2=late
       Requires:    []string{},       // Hard dependencies
       Suggests:    []string{},       // Soft suggestions
       Templates:   []string{"components/<component>"},
       Skills:      []string{"setup-<component>"},
       ClaudeMD:    []string{"<component>"},
   },
   ```

2. **Create Helm chart templates** at `templates/components/<component>/helm/`
   - `Chart.yaml.tmpl` with chart metadata
   - `values.yaml.tmpl` with default configuration
   - `templates/` with K8s resource templates

3. **Create Tiltfile entry** at `templates/components/<component>/tiltfile-entry.tmpl`
   - Function to set up the component in local development
   - Resource dependencies and port forwards

4. **Create CLAUDE.md section** at `templates/claude-md/<component>.md.tmpl`
   - Usage rules, patterns, and best practices

5. **Create skill template** at `templates/skills/setup-<component>.md.tmpl`
   - Step-by-step guide for adding the component to a service

6. **Update dependency resolution** if the component has special dependency requirements

7. **Write tests**
   - Component renders in golden file tests
   - Dependency resolution includes transitive deps
   - `iron add <component>` works

## Checklist

- [ ] Component defined in registry
- [ ] Helm chart templates created
- [ ] Tiltfile entry template created
- [ ] CLAUDE.md section added
- [ ] Skill template added
- [ ] Dependency chain is correct
- [ ] Tests pass
