{{/*
Expand the name of the chart.
*/}}
{{- define "strimzi-kafka.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "strimzi-kafka.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "strimzi-kafka.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "strimzi-kafka.labels" -}}
helm.sh/chart: {{ include "strimzi-kafka.chart" . }}
{{ include "strimzi-kafka.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "strimzi-kafka.selectorLabels" -}}
app.kubernetes.io/name: {{ include "strimzi-kafka.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Kafka cluster name
*/}}
{{- define "strimzi-kafka.clusterName" -}}
{{- .Values.kafka.name | default (printf "%s-kafka" .Release.Name) }}
{{- end }}

{{/*
Kafka bootstrap server
*/}}
{{- define "strimzi-kafka.bootstrapServer" -}}
{{- printf "%s-kafka-bootstrap.%s:9092" (include "strimzi-kafka.clusterName" .) .Values.namespace }}
{{- end }}
