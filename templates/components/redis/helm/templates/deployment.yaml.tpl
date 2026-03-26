apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ .Release.Name }}-redis
  labels:
    app.kubernetes.io/name: redis
    app.kubernetes.io/instance: {{ .Release.Name }}
spec:
  replicas: {{ .Values.replicaCount }}
  selector:
    matchLabels:
      app.kubernetes.io/name: redis
      app.kubernetes.io/instance: {{ .Release.Name }}
  template:
    metadata:
      labels:
        app.kubernetes.io/name: redis
        app.kubernetes.io/instance: {{ .Release.Name }}
    spec:
      containers:
        - name: redis
          image: "{{ .Values.image.repository }}:{{ .Values.image.tag }}"
          imagePullPolicy: {{ .Values.image.pullPolicy }}
          command:
            - redis-server
            - /etc/redis/redis.conf
          ports:
            - name: redis
              containerPort: 6379
              protocol: TCP
          livenessProbe:
            exec:
              command:
                - redis-cli
                - ping
            initialDelaySeconds: 10
            periodSeconds: 10
          readinessProbe:
            exec:
              command:
                - redis-cli
                - ping
            initialDelaySeconds: 5
            periodSeconds: 5
          resources:
            {{- toYaml .Values.resources | nindent 12 }}
          volumeMounts:
            - name: redis-config
              mountPath: /etc/redis
{{- if .Values.metrics.enabled }}
        - name: redis-exporter
          image: "{{ .Values.metrics.image.repository }}:{{ .Values.metrics.image.tag }}"
          imagePullPolicy: {{ .Values.metrics.image.pullPolicy }}
          ports:
            - name: metrics
              containerPort: 9121
              protocol: TCP
          env:
            - name: REDIS_ADDR
              value: "redis://localhost:6379"
          resources:
            {{- toYaml .Values.metrics.resources | nindent 12 }}
{{- end }}
      volumes:
        - name: redis-config
          configMap:
            name: {{ .Release.Name }}-redis-config
