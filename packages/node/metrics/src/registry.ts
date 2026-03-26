import client from 'prom-client';
import type { MetricsConfig } from './types';

/**
 * Creates and configures a Prometheus metrics registry.
 */
export function createMetricsRegistry(config: MetricsConfig): client.Registry {
  const registry = new client.Registry();

  const defaultLabels: Record<string, string> = {
    service: config.serviceName,
    ...(config.defaultLabels ?? {}),
  };

  registry.setDefaultLabels(defaultLabels);

  if (config.defaultMetrics !== false) {
    client.collectDefaultMetrics({ register: registry, prefix: config.prefix });
  }

  return registry;
}
