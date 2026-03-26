import type { AxiosInstance } from 'axios';
import type { Logger } from '@ironplate/logger';

/** A pre-configured Axios instance returned by createHttpClient. */
export type HttpClient = AxiosInstance;

/** Options for creating an HTTP client. */
export interface HttpClientOptions {
  /** A human-readable name for this client (used in logs). */
  name?: string;
  /** Base URL for all requests. */
  baseURL?: string;
  /** Request timeout in milliseconds. Defaults to 10 000. */
  timeout?: number;
  /** Maximum number of automatic retries on 5xx / network errors. Defaults to 2. */
  maxRetries?: number;
  /** Base delay in milliseconds for exponential backoff between retries. Defaults to 300. */
  retryDelayMs?: number;
  /** Default headers to include with every request. */
  headers?: Record<string, string>;
  /** Correlation ID to propagate. If omitted a new UUID is generated per request. */
  correlationId?: string;
  /** Logger instance. Falls back to creating a default logger. */
  logger?: Logger;
  /** Circuit breaker configuration. Omit to disable the circuit breaker. */
  circuitBreaker?: CircuitBreakerOptions;
}

/** Configuration for the simple circuit breaker. */
export interface CircuitBreakerOptions {
  /** Number of consecutive failures before opening the circuit. Defaults to 5. */
  failureThreshold?: number;
  /** Milliseconds to wait before transitioning from OPEN to HALF_OPEN. Defaults to 30 000. */
  resetTimeoutMs?: number;
  /** Maximum concurrent requests allowed in HALF_OPEN state. Defaults to 1. */
  halfOpenMaxRequests?: number;
}

/**
 * Normalised API error thrown by the HTTP client when a request fails.
 * Captures the HTTP status code and any response body data.
 */
export class ApiError extends Error {
  public readonly status: number;
  public readonly data?: Record<string, unknown>;

  constructor(message: string, status: number, data?: Record<string, unknown>) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
    this.data = data;

    // Maintain proper prototype chain for instanceof checks.
    Object.setPrototypeOf(this, ApiError.prototype);
  }
}
