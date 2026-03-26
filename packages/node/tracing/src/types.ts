export interface TracingConfig {
  /** Service name for trace spans */
  serviceName: string;
  /** Service version */
  serviceVersion?: string;
  /** OTLP endpoint URL (default: http://localhost:4318/v1/traces) */
  endpoint?: string;
  /** Whether to enable tracing (default: true) */
  enabled?: boolean;
  /** Sample rate between 0 and 1 (default: 1.0) */
  sampleRate?: number;
  /** Additional resource attributes */
  attributes?: Record<string, string>;
}
