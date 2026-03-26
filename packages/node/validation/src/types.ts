import type { z } from 'zod';

export interface ValidateOptions {
  /** Whether to strip unknown keys (default: true) */
  stripUnknown?: boolean;
  /** Custom error message prefix */
  errorPrefix?: string;
}

export type SchemaMap = {
  body?: z.ZodType;
  query?: z.ZodType;
  params?: z.ZodType;
};
