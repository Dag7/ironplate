import axios, { type AxiosInstance, type AxiosError, type InternalAxiosRequestConfig } from 'axios';
import { createLogger } from '@ironplate/logger';
import { ApiError } from './types';
import type { HttpClient, HttpClientOptions, CircuitBreakerOptions } from './types';

/** Possible circuit breaker states. */
enum CircuitState {
  Closed = 'CLOSED',
  Open = 'OPEN',
  HalfOpen = 'HALF_OPEN',
}

interface CircuitBreaker {
  state: CircuitState;
  failureCount: number;
  lastFailureTime: number;
  options: Required<CircuitBreakerOptions>;
}

const DEFAULT_TIMEOUT_MS = 10_000;
const DEFAULT_MAX_RETRIES = 2;
const DEFAULT_RETRY_DELAY_MS = 300;

/**
 * Creates a pre-configured axios HTTP client with request/response logging,
 * correlation ID propagation, retry logic, error normalisation, and an
 * optional circuit breaker.
 *
 * @param options - Client configuration.
 * @returns A configured Axios instance (HttpClient).
 */
export function createHttpClient(options: HttpClientOptions = {}): HttpClient {
  const logger = options.logger ?? createLogger(options.name ?? 'http-client');
  const maxRetries = options.maxRetries ?? DEFAULT_MAX_RETRIES;
  const retryDelay = options.retryDelayMs ?? DEFAULT_RETRY_DELAY_MS;

  const instance: AxiosInstance = axios.create({
    baseURL: options.baseURL,
    timeout: options.timeout ?? DEFAULT_TIMEOUT_MS,
    headers: {
      'Content-Type': 'application/json',
      ...(options.headers ?? {}),
    },
  });

  // ── Circuit breaker state ──────────────────────────────────────────
  let breaker: CircuitBreaker | undefined;
  if (options.circuitBreaker) {
    breaker = {
      state: CircuitState.Closed,
      failureCount: 0,
      lastFailureTime: 0,
      options: {
        failureThreshold: options.circuitBreaker.failureThreshold ?? 5,
        resetTimeoutMs: options.circuitBreaker.resetTimeoutMs ?? 30_000,
        halfOpenMaxRequests: options.circuitBreaker.halfOpenMaxRequests ?? 1,
      },
    };
  }

  // ── Request interceptor ────────────────────────────────────────────
  instance.interceptors.request.use((config: InternalAxiosRequestConfig) => {
    // Propagate correlation ID.
    const correlationId =
      config.headers?.['x-correlation-id'] ?? options.correlationId ?? crypto.randomUUID();
    config.headers.set('x-correlation-id', correlationId);

    // Circuit breaker guard.
    if (breaker) {
      if (breaker.state === CircuitState.Open) {
        const elapsed = Date.now() - breaker.lastFailureTime;
        if (elapsed >= breaker.options.resetTimeoutMs) {
          breaker.state = CircuitState.HalfOpen;
          logger.info('Circuit breaker transitioning to HALF_OPEN');
        } else {
          const err = new ApiError('Circuit breaker is OPEN — request blocked', 503);
          return Promise.reject(err);
        }
      }
    }

    logger.debug(
      { method: config.method?.toUpperCase(), url: config.url },
      'Outgoing request',
    );

    return config;
  });

  // ── Response interceptor ───────────────────────────────────────────
  instance.interceptors.response.use(
    (response) => {
      // Reset circuit breaker on success.
      if (breaker) {
        if (breaker.state === CircuitState.HalfOpen) {
          logger.info('Circuit breaker transitioning to CLOSED');
        }
        breaker.state = CircuitState.Closed;
        breaker.failureCount = 0;
      }

      logger.debug(
        {
          method: response.config.method?.toUpperCase(),
          url: response.config.url,
          status: response.status,
        },
        'Response received',
      );

      return response;
    },
    async (error: AxiosError) => {
      // Update circuit breaker on failure.
      if (breaker) {
        breaker.failureCount += 1;
        breaker.lastFailureTime = Date.now();
        if (breaker.failureCount >= breaker.options.failureThreshold) {
          breaker.state = CircuitState.Open;
          logger.warn(
            { failureCount: breaker.failureCount },
            'Circuit breaker transitioning to OPEN',
          );
        }
      }

      const config = error.config as InternalAxiosRequestConfig & { _retryCount?: number };
      const retryCount = config?._retryCount ?? 0;
      const isRetryable =
        error.response == null || // network error
        (error.response.status >= 500 && error.response.status < 600);

      if (isRetryable && retryCount < maxRetries && config) {
        config._retryCount = retryCount + 1;
        const delay = retryDelay * Math.pow(2, retryCount);
        logger.warn(
          { attempt: config._retryCount, maxRetries, delay, url: config.url },
          'Retrying request',
        );
        await new Promise((resolve) => setTimeout(resolve, delay));
        return instance.request(config);
      }

      // Normalise into ApiError.
      const status = error.response?.status ?? 0;
      const data = error.response?.data as Record<string, unknown> | undefined;
      const message =
        (data?.message as string) ?? error.message ?? 'Unknown HTTP error';

      logger.error(
        { method: config?.method?.toUpperCase(), url: config?.url, status, error: message },
        'Request failed',
      );

      throw new ApiError(message, status, data);
    },
  );

  return instance as HttpClient;
}
