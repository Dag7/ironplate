import { config as dotenvConfig } from 'dotenv';
import { z } from 'zod';
import type { ConfigOptions } from './types';

// Load .env files before anything reads process.env.
dotenvConfig();

/**
 * Base configuration schema shared by all services.
 * Provides NODE_ENV, LOG_LEVEL, PORT, and SERVICE_NAME.
 */
export const baseConfigSchema = z.object({
  /** Runtime environment. */
  NODE_ENV: z
    .enum(['development', 'production', 'test'])
    .default('development'),

  /** Logging level. */
  LOG_LEVEL: z
    .enum(['fatal', 'error', 'warn', 'info', 'debug', 'trace'])
    .default('info'),

  /** HTTP server port. */
  PORT: z.coerce.number().int().positive().default(3000),

  /** Logical service name used in logs and tracing. */
  SERVICE_NAME: z.string().min(1).default('unknown-service'),
});

/**
 * Creates a validated, typed configuration object from environment variables.
 *
 * The provided `schema` is merged with the base config schema so that every
 * service automatically gets NODE_ENV, LOG_LEVEL, PORT, and SERVICE_NAME
 * in addition to its own fields.
 *
 * @param options - Configuration loader options.
 * @returns A frozen, fully validated config object.
 *
 * @example
 * ```ts
 * const serviceSchema = z.object({
 *   DATABASE_URL: z.string().url(),
 *   REDIS_HOST: z.string().default('localhost'),
 * });
 *
 * const config = createConfig({ schema: serviceSchema });
 * // config.PORT, config.DATABASE_URL, etc. are all typed and validated.
 * ```
 */
export function createConfig<T extends z.ZodRawShape>(
  options: ConfigOptions<T> = {} as ConfigOptions<T>,
) {
  const mergedSchema = options.schema
    ? baseConfigSchema.merge(z.object(options.schema.shape))
    : baseConfigSchema;

  const envSource = options.env ?? process.env;

  const result = mergedSchema.safeParse(envSource);

  if (!result.success) {
    const formatted = result.error.issues
      .map((issue) => {
        const path = issue.path.join('.');
        return `  - ${path}: ${issue.message}`;
      })
      .join('\n');

    const message = `Configuration validation failed:\n${formatted}`;

    if (options.throwOnError === false) {
      console.error(message);
      return undefined;
    }

    throw new Error(message);
  }

  return Object.freeze(result.data) as z.infer<typeof mergedSchema>;
}
