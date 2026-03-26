apiVersion: v1
kind: Service
metadata:
  name: {{ .Release.Name }}-redis
  labels:
    app.kubernetes.io/name: redis
    app.kubernetes.io/instance: {{ .Release.Name }}
spec:
  type: {{ .Values.service.type }}
  ports:
    - port: {{ .Values.service.port }}
      targetPort: redis
      protocol: TCP
      name: redis
{{- if .Values.metrics.enabled }}
    - port: 9121
      targetPort: metrics
      protocol: TCP
      name: metrics
{{- end }}
  selector:
    app.kubernetes.io/name: redis
    app.kubernetes.io/instance: {{ .Release.Name }}
