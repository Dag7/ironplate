import { z } from 'zod';

/**
 * Common reusable Zod schemas for API validation.
 */

/** UUID v4 string */
export const uuid = z.string().uuid();

/** Non-empty trimmed string */
export const nonEmptyString = z.string().trim().min(1);

/** Email address */
export const email = z.string().email().toLowerCase();

/** URL string */
export const url = z.string().url();

/** ISO 8601 date string */
export const isoDate = z.string().datetime();

/** Positive integer */
export const positiveInt = z.coerce.number().int().positive();

/** Non-negative integer */
export const nonNegativeInt = z.coerce.number().int().nonnegative();

/** Pagination parameters */
export const pagination = z.object({
  page: z.coerce.number().int().positive().default(1),
  limit: z.coerce.number().int().positive().max(100).default(20),
});

/** Sort parameters */
export const sort = z.object({
  sortBy: z.string().default('createdAt'),
  sortOrder: z.enum(['asc', 'desc']).default('desc'),
});

/** Slug (URL-safe kebab-case string) */
export const slug = z.string().regex(/^[a-z0-9]+(?:-[a-z0-9]+)*$/, 'Invalid slug format');

/** Comma-separated list (parsed to array) */
export const commaSeparatedList = z
  .string()
  .transform((val) => val.split(',').map((s) => s.trim()).filter(Boolean));

/** Phone number (basic international format) */
export const phoneNumber = z.string().regex(/^\+?[1-9]\d{1,14}$/, 'Invalid phone number');

/** Hex color code */
export const hexColor = z.string().regex(/^#([0-9a-fA-F]{3}|[0-9a-fA-F]{6})$/, 'Invalid hex color');

/** Semantic version string */
export const semver = z.string().regex(
  /^\d+\.\d+\.\d+(-[a-zA-Z0-9.]+)?(\+[a-zA-Z0-9.]+)?$/,
  'Invalid semver',
);
