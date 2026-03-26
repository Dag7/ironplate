import { z } from 'zod';
import { ValidationError } from '@ironplate/errors';
import type { Request, Response, NextFunction } from 'express';
import type { SchemaMap, ValidateOptions } from './types';

/**
 * Creates Express middleware that validates request body, query, and params
 * against Zod schemas.
 *
 * @example
 * ```ts
 * router.post(
 *   '/users',
 *   validate({
 *     body: z.object({
 *       name: z.string().min(1),
 *       email: z.string().email(),
 *     }),
 *   }),
 *   createUser,
 * );
 * ```
 */
export function validate(schemas: SchemaMap, options?: ValidateOptions) {
  const stripUnknown = options?.stripUnknown ?? true;

  return (req: Request, _res: Response, next: NextFunction): void => {
    const errors: Record<string, string[]> = {};

    if (schemas.body) {
      const result = schemas.body.safeParse(req.body);
      if (!result.success) {
        errors.body = result.error.issues.map(formatIssue);
      } else if (stripUnknown) {
        req.body = result.data;
      }
    }

    if (schemas.query) {
      const result = schemas.query.safeParse(req.query);
      if (!result.success) {
        errors.query = result.error.issues.map(formatIssue);
      } else if (stripUnknown) {
        (req as any).query = result.data;
      }
    }

    if (schemas.params) {
      const result = schemas.params.safeParse(req.params);
      if (!result.success) {
        errors.params = result.error.issues.map(formatIssue);
      }
    }

    if (Object.keys(errors).length > 0) {
      const prefix = options?.errorPrefix ?? 'Validation failed';
      next(new ValidationError(prefix, { errors }));
      return;
    }

    next();
  };
}

function formatIssue(issue: z.ZodIssue): string {
  const path = issue.path.join('.');
  return path ? `${path}: ${issue.message}` : issue.message;
}
