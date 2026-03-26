{{/*
Expand the name of the chart.
*/}}
{{- define "service.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create chart name and version for labels.
*/}}
{{- define "service.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels for a service.
*/}}
{{- define "service.labels" -}}
{{- $svcName := index . 0 -}}
{{- $root := index . 1 -}}
helm.sh/chart: {{ include "service.chart" $root }}
app.kubernetes.io/name: {{ $svcName }}
app.kubernetes.io/instance: {{ $root.Release.Name }}
app.kubernetes.io/version: {{ $root.Chart.AppVersion | quote }}
app.kubernetes.io/managed-by: {{ $root.Release.Service }}
{{- with $root.Values.global.labels }}
{{ toYaml . }}
{{- end }}
{{- end }}

{{/*
Selector labels for a service.
*/}}
{{- define "service.selectorLabels" -}}
{{- $svcName := index . 0 -}}
{{- $root := index . 1 -}}
app.kubernetes.io/name: {{ $svcName }}
app.kubernetes.io/instance: {{ $root.Release.Name }}
{{- end }}

{{/*
Service account name for a service.
*/}}
{{- define "service.serviceAccountName" -}}
{{- $svcName := index . 0 -}}
{{- $config := index . 1 -}}
{{- if and $config.serviceAccount $config.serviceAccount.name -}}
{{- $config.serviceAccount.name }}
{{- else -}}
{{- $svcName }}
{{- end -}}
{{- end }}

{{/*
Construct the full image reference for a service.
Priority: service.image.repository > global.image.registry/svcName
*/}}
{{- define "service.image" -}}
{{- $svcName := index . 0 -}}
{{- $config := index . 1 -}}
{{- $root := index . 2 -}}
{{- $tag := "latest" -}}
{{- $repository := "" -}}
{{- if $config.image -}}
  {{- if $config.image.tag -}}
    {{- $tag = $config.image.tag -}}
  {{- end -}}
  {{- if $config.image.repository -}}
    {{- $repository = $config.image.repository -}}
  {{- else if $config.image.registry -}}
    {{- $repository = printf "%s/%s" $config.image.registry $svcName -}}
  {{- end -}}
{{- end -}}
{{- if not $repository -}}
  {{- $repository = printf "%s/%s" $root.Values.global.image.registry $svcName -}}
{{- end -}}
{{- printf "%s:%s" $repository $tag -}}
{{- end }}

{{/*
Get image pull policy.
*/}}
{{- define "service.imagePullPolicy" -}}
{{- $config := index . 0 -}}
{{- $root := index . 1 -}}
{{- if and $config.image $config.image.pullPolicy -}}
{{- $config.image.pullPolicy -}}
{{- else -}}
{{- $root.Values.global.image.pullPolicy | default "IfNotPresent" -}}
{{- end -}}
{{- end }}

{{/*
Get a value with defaults fallback.
*/}}
{{- define "service.value" -}}
{{- $config := index . 0 -}}
{{- $key := index . 1 -}}
{{- $defaults := index . 2 -}}
{{- if hasKey $config $key -}}
{{- index $config $key -}}
{{- else -}}
{{- index $defaults $key -}}
{{- end -}}
{{- end }}

{{/*
Merge service config with defaults.
Handles nested objects: resources, healthCheck, ingress, autoscaling, pdb, serviceAccount
*/}}
{{- define "service.mergedConfig" -}}
{{- $config := index . 0 -}}
{{- $defaults := index . 1 -}}
{{- $merged := dict -}}
{{- range $key, $value := $defaults -}}
  {{- if hasKey $config $key -}}
    {{- $_ := set $merged $key (index $config $key) -}}
  {{- else -}}
    {{- $_ := set $merged $key $value -}}
  {{- end -}}
{{- end -}}
{{- range $key, $value := $config -}}
  {{- if not (hasKey $merged $key) -}}
    {{- $_ := set $merged $key $value -}}
  {{- end -}}
{{- end -}}
{{- toYaml $merged -}}
{{- end }}

{{/*
Generate environment variables from commonEnv + service-specific env.
*/}}
{{- define "service.envVars" -}}
{{- $config := index . 0 -}}
{{- $root := index . 1 -}}
{{- $svcName := index . 2 -}}
{{- /* Common env from global */}}
{{- if $root.Values.global.commonEnv }}
{{- range $key, $value := $root.Values.global.commonEnv }}
- name: {{ $key }}
  value: {{ $value | quote }}
{{- end }}
{{- end }}
{{- /* Service-specific env */}}
{{- if $config.env }}
{{- range $key, $value := $config.env }}
- name: {{ $key }}
  value: {{ $value | quote }}
{{- end }}
{{- end }}
{{- /* Standard env vars */}}
- name: SERVICE_NAME
  value: {{ $svcName | quote }}
- name: ENVIRONMENT
  value: {{ $root.Values.global.environment | quote }}
- name: NAMESPACE
  valueFrom:
    fieldRef:
      fieldPath: metadata.namespace
- name: POD_NAME
  valueFrom:
    fieldRef:
      fieldPath: metadata.name
{{- /* Secret env vars with key remapping */}}
{{- if $config.secretEnv }}
{{- range $config.secretEnv }}
- name: {{ .name }}
  valueFrom:
    secretKeyRef:
      name: {{ .secretName }}
      key: {{ .secretKey }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Generate envFrom for secrets.
*/}}
{{- define "service.secretRefs" -}}
{{- $config := . -}}
{{- if $config.secrets }}
{{- range $config.secrets }}
- secretRef:
    name: {{ . }}
{{- end }}
{{- end }}
{{- if $config.configMaps }}
{{- range $config.configMaps }}
- configMapRef:
    name: {{ . }}
{{- end }}
{{- end }}
{{- end }}
