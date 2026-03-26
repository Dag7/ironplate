import type { Request, Response, NextFunction } from 'express';

export interface MetricsConfig {
  /** Service name label applied to all metrics */
  serviceName: string;
  /** Prefix for metric names (default: '') */
  prefix?: string;
  /** Include default Node.js metrics (default: true) */
  defaultMetrics?: boolean;
  /** Custom labels applied to all metrics */
  defaultLabels?: Record<string, string>;
  /** Paths to exclude from HTTP metrics (default: ['/healthz', '/readyz', '/metrics']) */
  excludePaths?: string[];
}

export interface HttpMetricsBuckets {
  /** Histogram buckets for request duration in seconds */
  durationBuckets?: number[];
  /** Histogram buckets for request size in bytes */
  sizeBuckets?: number[];
}
