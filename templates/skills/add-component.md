# Add Infrastructure Component

## Arguments
- `component`: Component name (e.g., `kafka`, `redis`, `hasura`, `dapr`, `observability`)

## Available Components
| Component | Description | Dependencies |
|-----------|-------------|--------------|
| kafka | Event streaming (Strimzi + KRaft) | helm-lib |
| hasura | GraphQL Engine + migrations | helm-lib |
| dapr | Distributed runtime (pub/sub, state) | helm-lib, kafka |
| redis | In-memory cache/store | helm-lib |
| observability | OTEL + Tempo + Prometheus + Grafana | helm-lib |
| external-secrets | Cloud secret sync to K8s | helm-lib |
| argocd | GitOps continuous delivery | helm-lib, external-secrets |
| langfuse | LLM observability | helm-lib, redis |

## Steps

### 1. Resolve Dependencies
- Check component dependency chain
- Auto-install required dependencies (e.g., `dapr` requires `kafka`)

### 2. Install Helm Chart
- Create `k8s/helm/infra/{component}/`
- Add Chart.yaml, values.yaml, templates

### 3. Update Tiltfile
- Add component setup in `IRONPLATE:INFRA:START/END` section
- Add `setup_{component}("development")` call

### 4. Update Configuration
- Update `ironplate.yaml` to include new component
- Regenerate CLAUDE.md sections
- Add component-specific skills if available

## Checklist
- [ ] Dependencies resolved and installed
- [ ] Helm chart created in `k8s/helm/infra/{component}/`
- [ ] Tiltfile updated
- [ ] `ironplate.yaml` updated
- [ ] CLAUDE.md regenerated
- [ ] `tilt up` shows the new component
