import client from 'prom-client';
import type { Request, Response, NextFunction } from 'express';
import type { MetricsConfig, HttpMetricsBuckets } from './types';

const DEFAULT_DURATION_BUCKETS = [0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10];

/**
 * Creates Express middleware that records RED (Rate, Errors, Duration) metrics
 * for HTTP requests.
 */
export function createHttpMetrics(
  registry: client.Registry,
  config: MetricsConfig,
  buckets?: HttpMetricsBuckets,
) {
  const prefix = config.prefix ?? '';
  const excludePaths = new Set(
    config.excludePaths ?? ['/healthz', '/readyz', '/metrics'],
  );

  const requestDuration = new client.Histogram({
    name: `${prefix}http_request_duration_seconds`,
    help: 'Duration of HTTP requests in seconds',
    labelNames: ['method', 'route', 'status_code'] as const,
    buckets: buckets?.durationBuckets ?? DEFAULT_DURATION_BUCKETS,
    registers: [registry],
  });

  const requestTotal = new client.Counter({
    name: `${prefix}http_requests_total`,
    help: 'Total number of HTTP requests',
    labelNames: ['method', 'route', 'status_code'] as const,
    registers: [registry],
  });

  const requestErrors = new client.Counter({
    name: `${prefix}http_request_errors_total`,
    help: 'Total number of HTTP request errors (status >= 400)',
    labelNames: ['method', 'route', 'status_code'] as const,
    registers: [registry],
  });

  const middleware = (req: Request, res: Response, next: NextFunction): void => {
    if (excludePaths.has(req.path)) {
      return next();
    }

    const start = process.hrtime.bigint();

    res.on('finish', () => {
      const durationNs = Number(process.hrtime.bigint() - start);
      const durationSec = durationNs / 1e9;
      const route = req.route?.path ?? req.path;
      const labels = {
        method: req.method,
        route,
        status_code: String(res.statusCode),
      };

      requestDuration.observe(labels, durationSec);
      requestTotal.inc(labels);

      if (res.statusCode >= 400) {
        requestErrors.inc(labels);
      }
    });

    next();
  };

  return { middleware, requestDuration, requestTotal, requestErrors };
}
