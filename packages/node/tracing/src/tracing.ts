import { NodeSDK } from '@opentelemetry/sdk-node';
import { OTLPTraceExporter } from '@opentelemetry/exporter-trace-otlp-http';
import { Resource } from '@opentelemetry/resources';
import { ATTR_SERVICE_NAME, ATTR_SERVICE_VERSION } from '@opentelemetry/semantic-conventions';
import { HttpInstrumentation } from '@opentelemetry/instrumentation-http';
import { ExpressInstrumentation } from '@opentelemetry/instrumentation-express';
import type { TracingConfig } from './types';

let sdk: NodeSDK | undefined;

/**
 * Initializes OpenTelemetry tracing for the service.
 *
 * IMPORTANT: Call this BEFORE importing Express or any HTTP libraries
 * to ensure instrumentation hooks are properly installed.
 *
 * @example
 * ```ts
 * // At the very top of your entry file:
 * import { initTracing } from '@ironplate/tracing';
 *
 * initTracing({
 *   serviceName: 'user-api',
 *   endpoint: process.env.OTEL_ENDPOINT,
 * });
 *
 * // Then import everything else
 * import { createService } from '@ironplate/service';
 * ```
 */
export function initTracing(config: TracingConfig): void {
  if (config.enabled === false) return;
  if (sdk) return; // Already initialized

  const resource = new Resource({
    [ATTR_SERVICE_NAME]: config.serviceName,
    [ATTR_SERVICE_VERSION]: config.serviceVersion ?? '0.0.0',
    ...(config.attributes ?? {}),
  });

  const exporter = new OTLPTraceExporter({
    url: config.endpoint ?? process.env.OTEL_EXPORTER_OTLP_ENDPOINT ?? 'http://localhost:4318/v1/traces',
  });

  sdk = new NodeSDK({
    resource,
    traceExporter: exporter,
    instrumentations: [
      new HttpInstrumentation(),
      new ExpressInstrumentation(),
    ],
  });

  sdk.start();

  // Graceful shutdown
  const shutdown = async () => {
    if (sdk) {
      await sdk.shutdown();
    }
  };
  process.on('SIGINT', shutdown);
  process.on('SIGTERM', shutdown);
}

/**
 * Shuts down the tracing SDK. Call this during graceful shutdown.
 */
export async function shutdownTracing(): Promise<void> {
  if (sdk) {
    await sdk.shutdown();
    sdk = undefined;
  }
}
