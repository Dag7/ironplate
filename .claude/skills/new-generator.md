# Add a New Generator Command

Add a new `iron generate <type>` subcommand.

## Arguments

- `name`: Generator name (e.g., "migration", "helm-chart")

## Steps

1. **Create CLI command** in `internal/cli/generate.go`
   - Add `newGenerate<Name>Cmd()` function returning `*cobra.Command`
   - Define flags for required parameters
   - Register as subcommand of `newGenerateCmd()`

2. **Create generator logic** at `internal/generators/<category>/<name>.go`
   - Implement generation logic using `engine.Renderer`
   - Load templates from embedded filesystem
   - Handle file creation and existing file updates

3. **Create templates** at `templates/<category>/<name>/`
   - Add all `.tmpl` files the generator needs

4. **Handle project updates**
   - If the generator modifies existing files (Tiltfile, ironplate.yaml), use marked sections:
     ```
     # IRONPLATE:<SECTION>:START
     # ... generated content ...
     # IRONPLATE:<SECTION>:END
     ```
   - Parse existing file, replace content between markers

5. **Write tests**
   - Unit test for generator logic
   - Golden file test for template output
   - Integration test that runs the full command

## Checklist

- [ ] CLI command registered in `generate.go`
- [ ] Generator logic handles both new and existing projects
- [ ] Templates render correctly
- [ ] Existing files updated via marked sections (if applicable)
- [ ] Tests pass
- [ ] Help text and examples included
