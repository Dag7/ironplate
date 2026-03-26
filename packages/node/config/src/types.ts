import type { z } from 'zod';
import type { baseConfigSchema } from './config';

/** Inferred type of the base configuration schema. */
export type BaseConfig = z.infer<typeof baseConfigSchema>;

/** Options accepted by createConfig. */
export interface ConfigOptions<T extends z.ZodRawShape = z.ZodRawShape> {
  /**
   * Service-specific zod object schema to merge with the base schema.
   * All fields from the base schema (NODE_ENV, LOG_LEVEL, PORT, SERVICE_NAME)
   * are included automatically.
   */
  schema?: z.ZodObject<T>;

  /**
   * Custom environment source. Defaults to process.env.
   * Useful for testing with a fixed set of variables.
   */
  env?: Record<string, string | undefined>;

  /**
   * When false, validation errors are logged to stderr and the function
   * returns undefined instead of throwing. Defaults to true (throw).
   */
  throwOnError?: boolean;
}
