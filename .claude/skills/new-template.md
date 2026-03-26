# Add a New Template Category

Add a new template category to the ironplate template system.

## Arguments

- `category`: Template category name (e.g., "monitoring", "messaging")

## Steps

1. **Create template directory** at `templates/<category>/`
   - Add `.tmpl` files for each generated file
   - Use `{{ .Project.Metadata.Name }}` for project name references
   - Use `{{ .Computed.Has* }}` flags for conditional sections

2. **Register in component registry** (`internal/components/registry.go`)
   - Add entry to `builtinComponents` map
   - Set `Name`, `Description`, `Tier`, `Requires`, `Templates`
   - Define `Skills` and `ClaudeMD` sections if applicable

3. **Add CLAUDE.md section** (if needed) at `templates/claude-md/<category>.md.tmpl`

4. **Add skill template** (if needed) at `templates/skills/setup-<category>.md.tmpl`

5. **Write golden file test** in `testdata/golden/`
   - Create input config that includes the new component
   - Generate expected output directory tree

6. **Update `iron list components`** output to include the new category

## Checklist

- [ ] Template directory created with all `.tmpl` files
- [ ] Component registered in `internal/components/registry.go`
- [ ] Dependencies declared correctly (`Requires`, `Suggests`)
- [ ] CLAUDE.md section template added (if applicable)
- [ ] Skill template added (if applicable)
- [ ] Golden file test passes
- [ ] `iron add <category>` works end-to-end
