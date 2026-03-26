import type pino from 'pino';

/** A structured logger instance. */
export type Logger = pino.Logger;

/** Context fields that can be bound to a child logger. */
export interface LogContext {
  /** Distributed trace correlation ID. */
  correlationId?: string;
  /** Unique identifier for a single request. */
  requestId?: string;
  /** Additional arbitrary context fields. */
  [key: string]: unknown;
}

/** Options for creating a logger. */
export interface LoggerOptions {
  /** Log level override. Defaults to LOG_LEVEL env var or 'info'. */
  level?: string;
  /** Additional field paths to redact from log output. */
  redactPaths?: string[];
  /** Raw pino options to merge with defaults. */
  pinoOptions?: Record<string, unknown>;
}
