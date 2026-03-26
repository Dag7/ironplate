apiVersion: v1
kind: ConfigMap
metadata:
  name: {{ .Release.Name }}-redis-config
  labels:
    app.kubernetes.io/name: redis
    app.kubernetes.io/instance: {{ .Release.Name }}
data:
  redis.conf: |
    bind 0.0.0.0
    protected-mode no
    port 6379
    tcp-backlog 511
    timeout 0
    tcp-keepalive 300
    maxmemory {{ .Values.redis.maxmemory }}
    maxmemory-policy {{ .Values.redis.maxmemoryPolicy }}
    appendonly no
    save ""
    loglevel notice
