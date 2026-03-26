{{/*
=============================================================================
Service Template - Renders all K8s resources for a microservice
=============================================================================
Called from umbrella charts. Iterates over .Values.services and generates:
- ServiceAccount
- Deployment
- Service
- Ingress (optional)
- HPA (optional)
- PDB (optional)
=============================================================================
*/}}
{{- define "service.render" -}}
{{- $root := . -}}
{{- $defaults := .Values.defaults -}}

{{- range $svcName, $svcConfig := .Values.services }}
{{- if ne (default true $svcConfig.enabled) false }}

{{- /* Merge service config with defaults */}}
{{- $replicas := default $defaults.replicas $svcConfig.replicas }}
{{- $port := default $defaults.port $svcConfig.port }}
{{- $serviceType := default $defaults.serviceType $svcConfig.serviceType }}

{{- /* ServiceAccount */}}
{{- $saCreate := true }}
{{- if $svcConfig.serviceAccount }}
  {{- if hasKey $svcConfig.serviceAccount "create" }}
    {{- $saCreate = $svcConfig.serviceAccount.create }}
  {{- end }}
{{- else if $defaults.serviceAccount }}
  {{- $saCreate = $defaults.serviceAccount.create }}
{{- end }}

{{- if $saCreate }}
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: {{ $svcName }}
  namespace: {{ $root.Values.global.namespace }}
  labels:
    {{- include "service.labels" (list $svcName $root) | nindent 4 }}
{{- end }}

{{/* ===== Deployment ===== */}}
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ $svcName }}
  namespace: {{ $root.Values.global.namespace }}
  labels:
    {{- include "service.labels" (list $svcName $root) | nindent 4 }}
  {{- with $svcConfig.annotations }}
  annotations:
    {{- toYaml . | nindent 4 }}
  {{- end }}
spec:
  {{- if not (and $svcConfig.autoscaling $svcConfig.autoscaling.enabled) }}
  replicas: {{ $replicas }}
  {{- end }}
  selector:
    matchLabels:
      {{- include "service.selectorLabels" (list $svcName $root) | nindent 6 }}
  template:
    metadata:
      labels:
        {{- include "service.selectorLabels" (list $svcName $root) | nindent 8 }}
        {{- with $root.Values.global.labels }}
        {{- toYaml . | nindent 8 }}
        {{- end }}
      {{- with $svcConfig.podAnnotations }}
      annotations:
        {{- toYaml . | nindent 8 }}
      {{- end }}
    spec:
      {{- if $saCreate }}
      serviceAccountName: {{ $svcName }}
      {{- end }}

      {{- /* Pod security context */}}
      {{- $podSecCtx := $defaults.podSecurityContext }}
      {{- if $svcConfig.podSecurityContext }}
        {{- $podSecCtx = $svcConfig.podSecurityContext }}
      {{- end }}
      {{- with $podSecCtx }}
      securityContext:
        {{- toYaml . | nindent 8 }}
      {{- end }}

      {{- /* Init containers */}}
      {{- with $svcConfig.initContainers }}
      initContainers:
        {{- toYaml . | nindent 8 }}
      {{- end }}

      containers:
        - name: {{ $svcName }}
          image: {{ include "service.image" (list $svcName $svcConfig $root) }}
          imagePullPolicy: {{ include "service.imagePullPolicy" (list $svcConfig $root) }}

          {{- /* Container security context */}}
          {{- $secCtx := $defaults.securityContext }}
          {{- if $svcConfig.securityContext }}
            {{- $secCtx = $svcConfig.securityContext }}
          {{- end }}
          {{- with $secCtx }}
          securityContext:
            {{- toYaml . | nindent 12 }}
          {{- end }}

          ports:
            - name: http
              containerPort: {{ $port }}
              protocol: TCP
            {{- with $svcConfig.additionalPorts }}
            {{- toYaml . | nindent 12 }}
            {{- end }}

          {{- /* Environment variables */}}
          env:
            {{- include "service.envVars" (list $svcConfig $root $svcName) | nindent 12 }}

          {{- /* EnvFrom (secrets + configmaps) */}}
          {{- $envFrom := include "service.secretRefs" $svcConfig }}
          {{- if $envFrom }}
          envFrom:
            {{- $envFrom | nindent 12 }}
          {{- end }}

          {{- /* Resources */}}
          {{- $resources := $defaults.resources }}
          {{- if $svcConfig.resources }}
            {{- $resources = $svcConfig.resources }}
          {{- end }}
          {{- with $resources }}
          resources:
            {{- toYaml . | nindent 12 }}
          {{- end }}

          {{- /* Health checks */}}
          {{- $hc := $defaults.healthCheck }}
          {{- if $svcConfig.healthCheck }}
            {{- $hc = merge $svcConfig.healthCheck $defaults.healthCheck }}
          {{- end }}
          {{- if $hc.enabled }}
          readinessProbe:
            httpGet:
              path: {{ $hc.readinessPath | default "/healthz" }}
              port: http
            initialDelaySeconds: {{ $hc.initialDelaySeconds | default 60 }}
            periodSeconds: {{ $hc.periodSeconds | default 30 }}
            timeoutSeconds: {{ $hc.timeoutSeconds | default 10 }}
            failureThreshold: {{ $hc.failureThreshold | default 10 }}
          livenessProbe:
            httpGet:
              path: {{ $hc.livenessPath | default "/healthz" }}
              port: http
            initialDelaySeconds: {{ $hc.initialDelaySeconds | default 60 }}
            periodSeconds: {{ $hc.periodSeconds | default 30 }}
            timeoutSeconds: {{ $hc.timeoutSeconds | default 10 }}
            failureThreshold: {{ $hc.failureThreshold | default 10 }}
          {{- end }}

          {{- /* Volume mounts */}}
          {{- with $svcConfig.volumeMounts }}
          volumeMounts:
            {{- toYaml . | nindent 12 }}
          {{- end }}

      {{- /* Volumes */}}
      {{- with $svcConfig.volumes }}
      volumes:
        {{- toYaml . | nindent 8 }}
      {{- end }}

      {{- /* Node scheduling */}}
      {{- $nodeSelector := default $defaults.nodeSelector $svcConfig.nodeSelector }}
      {{- with $nodeSelector }}
      nodeSelector:
        {{- toYaml . | nindent 8 }}
      {{- end }}
      {{- $tolerations := default $defaults.tolerations $svcConfig.tolerations }}
      {{- with $tolerations }}
      tolerations:
        {{- toYaml . | nindent 8 }}
      {{- end }}
      {{- $affinity := default $defaults.affinity $svcConfig.affinity }}
      {{- with $affinity }}
      affinity:
        {{- toYaml . | nindent 8 }}
      {{- end }}

{{/* ===== Service ===== */}}
---
apiVersion: v1
kind: Service
metadata:
  name: {{ $svcName }}
  namespace: {{ $root.Values.global.namespace }}
  labels:
    {{- include "service.labels" (list $svcName $root) | nindent 4 }}
spec:
  type: {{ $serviceType }}
  ports:
    - port: 80
      targetPort: {{ $port }}
      protocol: TCP
      name: http
    {{- with $svcConfig.additionalPorts }}
    {{- range . }}
    - port: {{ .containerPort }}
      targetPort: {{ .containerPort }}
      protocol: {{ .protocol | default "TCP" }}
      name: {{ .name }}
    {{- end }}
    {{- end }}
  selector:
    {{- include "service.selectorLabels" (list $svcName $root) | nindent 4 }}

{{- /* ===== Ingress ===== */}}
{{- $ing := $defaults.ingress }}
{{- if $svcConfig.ingress }}
  {{- $ing = merge $svcConfig.ingress $defaults.ingress }}
{{- end }}
{{- if $ing.enabled }}
---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: {{ $svcName }}
  namespace: {{ $root.Values.global.namespace }}
  labels:
    {{- include "service.labels" (list $svcName $root) | nindent 4 }}
  annotations:
    {{- with $ing.annotations }}
    {{- toYaml . | nindent 4 }}
    {{- end }}
    {{- if $ing.middlewares }}
    traefik.ingress.kubernetes.io/router.middlewares: {{ join "," $ing.middlewares }}
    {{- end }}
spec:
  ingressClassName: {{ $ing.className | default "traefik" }}
  {{- if $root.Values.global.tlsSecretName }}
  tls:
    - hosts:
        - {{ $ing.host }}
      secretName: {{ $root.Values.global.tlsSecretName }}
  {{- end }}
  rules:
    - host: {{ $ing.host }}
      http:
        paths:
          {{- if $ing.paths }}
          {{- range $ing.paths }}
          - path: {{ .path }}
            pathType: {{ .pathType | default "Prefix" }}
            backend:
              service:
                name: {{ .serviceName | default $svcName }}
                port:
                  number: {{ .port | default 80 }}
          {{- end }}
          {{- else }}
          - path: /
            pathType: Prefix
            backend:
              service:
                name: {{ $svcName }}
                port:
                  number: 80
          {{- end }}
{{- end }}

{{- /* ===== HPA ===== */}}
{{- $hpa := $defaults.autoscaling }}
{{- if $svcConfig.autoscaling }}
  {{- $hpa = merge $svcConfig.autoscaling $defaults.autoscaling }}
{{- end }}
{{- if $hpa.enabled }}
---
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: {{ $svcName }}
  namespace: {{ $root.Values.global.namespace }}
  labels:
    {{- include "service.labels" (list $svcName $root) | nindent 4 }}
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: {{ $svcName }}
  minReplicas: {{ $hpa.minReplicas | default 1 }}
  maxReplicas: {{ $hpa.maxReplicas | default 5 }}
  metrics:
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: {{ $hpa.targetCPUUtilizationPercentage | default 80 }}
    {{- if $hpa.targetMemoryUtilizationPercentage }}
    - type: Resource
      resource:
        name: memory
        target:
          type: Utilization
          averageUtilization: {{ $hpa.targetMemoryUtilizationPercentage }}
    {{- end }}
{{- end }}

{{- /* ===== PDB ===== */}}
{{- $pdb := $defaults.pdb }}
{{- if $svcConfig.pdb }}
  {{- $pdb = merge $svcConfig.pdb $defaults.pdb }}
{{- end }}
{{- if $pdb.enabled }}
---
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: {{ $svcName }}
  namespace: {{ $root.Values.global.namespace }}
  labels:
    {{- include "service.labels" (list $svcName $root) | nindent 4 }}
spec:
  selector:
    matchLabels:
      {{- include "service.selectorLabels" (list $svcName $root) | nindent 6 }}
  {{- if $pdb.minAvailable }}
  minAvailable: {{ $pdb.minAvailable }}
  {{- else if $pdb.maxUnavailable }}
  maxUnavailable: {{ $pdb.maxUnavailable }}
  {{- else }}
  minAvailable: 1
  {{- end }}
{{- end }}

{{- /* ===== Dapr Subscriptions ===== */}}
{{- if $svcConfig.daprSubscriptions }}
{{- range $svcConfig.daprSubscriptions }}
---
apiVersion: dapr.io/v2alpha1
kind: Subscription
metadata:
  name: {{ $svcName }}-{{ .topic }}
  namespace: {{ $root.Values.global.namespace }}
  labels:
    {{- include "service.labels" (list $svcName $root) | nindent 4 }}
spec:
  pubsubname: {{ .pubsubname | default "kafka-pubsub" }}
  topic: {{ .topic }}
  routes:
    default: {{ .route | default (printf "/events/%s" .topic) }}
  {{- if .deadLetterTopic }}
  deadLetterTopic: {{ .deadLetterTopic }}
  {{- end }}
  {{- if .bulkSubscribe }}
  bulkSubscribe:
    enabled: {{ .bulkSubscribe.enabled | default true }}
    maxMessagesCount: {{ .bulkSubscribe.maxMessagesCount | default 100 }}
    maxAwaitDurationMs: {{ .bulkSubscribe.maxAwaitDurationMs | default 1000 }}
  {{- end }}
{{- end }}
{{- end }}

{{- /* ===== RBAC ===== */}}
{{- if $svcConfig.rbac }}
{{- if $svcConfig.rbac.enabled }}
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: {{ $svcName }}
  namespace: {{ $root.Values.global.namespace }}
  labels:
    {{- include "service.labels" (list $svcName $root) | nindent 4 }}
rules:
  {{- toYaml $svcConfig.rbac.rules | nindent 2 }}
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: {{ $svcName }}
  namespace: {{ $root.Values.global.namespace }}
  labels:
    {{- include "service.labels" (list $svcName $root) | nindent 4 }}
subjects:
  - kind: ServiceAccount
    name: {{ $svcName }}
    namespace: {{ $root.Values.global.namespace }}
roleRef:
  kind: Role
  name: {{ $svcName }}
  apiGroup: rbac.authorization.k8s.io
{{- end }}
{{- end }}

{{- end }}{{/* if enabled */}}
{{- end }}{{/* range services */}}
{{- end }}{{/* define */}}
