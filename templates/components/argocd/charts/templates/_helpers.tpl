{{/*
=============================================================================
Helper Templates for ArgoCD App-of-Apps Chart
=============================================================================
*/}}

{{/*
Generate the full application name: <group>-<environment>
*/}}
{{- define "apps.appName" -}}
{{- printf "%s-%s" .name .environment -}}
{{- end -}}

{{/*
Generate the image registry path based on provider
*/}}
{{- define "apps.registry" -}}
{{- $global := .global -}}
{{- if eq $global.provider "gcp" -}}
{{- printf "%s-docker.pkg.dev/%s/%s" $global.region $global.project .projectName -}}
{{- else if eq $global.provider "aws" -}}
{{- printf "%s/%s" $global.imageUpdater.registryPrefix .projectName -}}
{{- else -}}
{{- printf "%s/%s" $global.imageUpdater.registryPrefix .projectName -}}
{{- end -}}
{{- end -}}

{{/*
Generate common labels
*/}}
{{- define "apps.labels" -}}
app.kubernetes.io/part-of: {{ .projectName }}
app.kubernetes.io/managed-by: argocd-apps-chart
environment: {{ .environment }}
{{- end -}}

{{/*
Generate sync policy
*/}}
{{- define "apps.syncPolicy" -}}
automated:
  prune: true
  selfHeal: true
syncOptions:
  - CreateNamespace=true
  - PrunePropagationPolicy=foreground
  - RespectIgnoreDifferences=true
{{- if eq .environment "production" }}
  - PruneLast=true
{{- end }}
retry:
  limit: 5
  backoff:
    duration: 5s
    factor: 2
    maxDuration: 3m
{{- end -}}
