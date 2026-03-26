import type { Router } from 'express';

export interface HealthCheck {
  name: string;
  check: () => Promise<HealthCheckResult>;
}

export interface HealthCheckResult {
  status: 'healthy' | 'unhealthy' | 'degraded';
  message?: string;
  latencyMs?: number;
}

export interface HealthResponse {
  status: 'ok' | 'degraded' | 'unhealthy';
  service: string;
  uptime: number;
  checks?: Record<string, HealthCheckResult>;
}

export interface HealthRouterOptions {
  /** Service name shown in response */
  serviceName: string;
  /** Custom health checks to run on /readyz */
  checks?: HealthCheck[];
  /** Path for liveness probe (default: /healthz) */
  livenessPath?: string;
  /** Path for readiness probe (default: /readyz) */
  readinessPath?: string;
}
