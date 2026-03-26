import pino from 'pino';
import type { Logger, LoggerOptions, LogContext } from './types';

const DEFAULT_REDACT_PATHS = [
  'password',
  'token',
  'authorization',
  'secret',
  'credentials',
  '*.password',
  '*.token',
  '*.authorization',
  '*.secret',
  '*.credentials',
  'req.headers.authorization',
  'req.headers.cookie',
];

/**
 * Creates a structured logger instance using pino.
 *
 * @param name - The logger name, typically the service or module name.
 * @param options - Optional logger configuration overrides.
 * @returns A configured pino Logger instance.
 */
export function createLogger(name: string, options: LoggerOptions = {}): Logger {
  const isDevelopment = process.env.NODE_ENV === 'development';
  const level = options.level ?? process.env.LOG_LEVEL ?? (isDevelopment ? 'debug' : 'info');

  const redact = {
    paths: [...DEFAULT_REDACT_PATHS, ...(options.redactPaths ?? [])],
    censor: '[REDACTED]',
  };

  const transport = isDevelopment
    ? {
        target: 'pino-pretty',
        options: {
          colorize: true,
          translateTime: 'SYS:standard',
          ignore: 'pid,hostname',
        },
      }
    : undefined;

  const baseLogger = pino({
    name,
    level,
    redact,
    transport,
    timestamp: pino.stdTimeFunctions.isoTime,
    formatters: {
      level(label: string) {
        return { level: label };
      },
    },
    ...(options.pinoOptions ?? {}),
  });

  return baseLogger;
}

/**
 * Creates a child logger with additional context bound to every log entry.
 * Useful for adding request-scoped fields like correlationId or requestId.
 *
 * @param logger - The parent logger instance.
 * @param context - Context fields to bind to the child logger.
 * @returns A child Logger with the given context.
 */
export function createChildLogger(logger: Logger, context: LogContext): Logger {
  return logger.child(context);
}

/**
 * Creates a request-scoped child logger with correlation and request IDs.
 *
 * @param logger - The parent logger instance.
 * @param correlationId - The correlation ID for distributed tracing.
 * @param requestId - An optional unique request identifier.
 * @returns A child Logger bound with request context.
 */
export function createRequestLogger(
  logger: Logger,
  correlationId: string,
  requestId?: string,
): Logger {
  return logger.child({
    correlationId,
    ...(requestId ? { requestId } : {}),
  });
}
