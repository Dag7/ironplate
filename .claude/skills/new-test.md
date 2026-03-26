# Generate Test Scaffolding

Generate test files for an existing package or feature.

## Arguments

- `package-path`: Go package path (e.g., "internal/config", "internal/engine")
- `test-type`: Test type: `unit`, `golden`, `integration`

## Steps

### Unit Test

1. Create `<package>/<file>_test.go` with table-driven test skeleton:
   ```go
   func Test<Function>(t *testing.T) {
       tests := []struct {
           name    string
           input   <type>
           want    <type>
           wantErr bool
       }{
           {"valid input", ..., ..., false},
           {"empty input", ..., ..., true},
       }
       for _, tt := range tests {
           t.Run(tt.name, func(t *testing.T) {
               got, err := <Function>(tt.input)
               if tt.wantErr {
                   require.Error(t, err)
                   return
               }
               require.NoError(t, err)
               assert.Equal(t, tt.want, got)
           })
       }
   }
   ```

### Golden File Test

1. Create `testdata/golden/<test-name>/input.yaml` with config fixture
2. Create `testdata/golden/<test-name>/expected/` with expected output tree
3. Add test function:
   ```go
   func TestGolden_<Name>(t *testing.T) {
       cfg := loadTestConfig(t, "testdata/golden/<test-name>/input.yaml")
       outputDir := t.TempDir()
       // Run scaffolder
       // Compare output with expected/
   }
   ```

### Integration Test

1. Create `internal/integration_test/<name>_test.go` with build tag:
   ```go
   //go:build integration
   ```
2. Run `iron` commands via `exec.Command`
3. Verify output structure and content

## Checklist

- [ ] Test file created in correct location
- [ ] Table-driven format used for unit tests
- [ ] Golden files created for template tests
- [ ] Build tag added for integration tests
- [ ] `make test` passes
