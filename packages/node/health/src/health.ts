import { Router } from 'express';
import type { HealthCheck, HealthCheckResult, HealthResponse, HealthRouterOptions } from './types';

const startTime = Date.now();

/**
 * Creates an Express router with Kubernetes-compatible health check endpoints.
 *
 * - Liveness probe (/healthz): Always returns 200 if the process is alive.
 * - Readiness probe (/readyz): Runs registered health checks and returns
 *   aggregate status.
 */
export function createHealthRouter(options: HealthRouterOptions): Router {
  const {
    serviceName,
    checks = [],
    livenessPath = '/healthz',
    readinessPath = '/readyz',
  } = options;

  const router = Router();

  router.get(livenessPath, (_req, res) => {
    const response: HealthResponse = {
      status: 'ok',
      service: serviceName,
      uptime: Math.floor((Date.now() - startTime) / 1000),
    };
    res.json(response);
  });

  router.get(readinessPath, async (_req, res) => {
    const results: Record<string, HealthCheckResult> = {};
    let overallStatus: HealthResponse['status'] = 'ok';

    for (const check of checks) {
      const start = Date.now();
      try {
        const result = await check.check();
        results[check.name] = {
          ...result,
          latencyMs: Date.now() - start,
        };
        if (result.status === 'unhealthy') {
          overallStatus = 'unhealthy';
        } else if (result.status === 'degraded' && overallStatus === 'ok') {
          overallStatus = 'degraded';
        }
      } catch (error) {
        results[check.name] = {
          status: 'unhealthy',
          message: error instanceof Error ? error.message : 'Unknown error',
          latencyMs: Date.now() - start,
        };
        overallStatus = 'unhealthy';
      }
    }

    const response: HealthResponse = {
      status: overallStatus,
      service: serviceName,
      uptime: Math.floor((Date.now() - startTime) / 1000),
      ...(Object.keys(results).length > 0 ? { checks: results } : {}),
    };

    const statusCode = overallStatus === 'unhealthy' ? 503 : 200;
    res.status(statusCode).json(response);
  });

  return router;
}
