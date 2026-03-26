import type client from 'prom-client';
import { Router } from 'express';

/**
 * Creates an Express router that serves Prometheus metrics at GET /metrics.
 */
export function createMetricsEndpoint(registry: client.Registry, path = '/metrics'): Router {
  const router = Router();

  router.get(path, async (_req, res) => {
    try {
      res.set('Content-Type', registry.contentType);
      const metrics = await registry.metrics();
      res.end(metrics);
    } catch (err) {
      res.status(500).end(String(err));
    }
  });

  return router;
}
