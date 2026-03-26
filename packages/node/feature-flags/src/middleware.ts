import type { Request, Response, NextFunction } from 'express';
import type { FeatureFlagClient } from './client';
import type { FeatureFlagUser } from './types';

/**
 * Express middleware that gates a route behind a feature flag.
 *
 * @example
 * ```ts
 * router.get(
 *   '/beta/dashboard',
 *   requireFlag(flags, 'beta-dashboard', (req) => ({ userId: req.auth.userId })),
 *   handleBetaDashboard,
 * );
 * ```
 */
export function requireFlag(
  client: FeatureFlagClient,
  flag: string,
  userExtractor?: (req: Request) => FeatureFlagUser,
) {
  return async (req: Request, res: Response, next: NextFunction): Promise<void> => {
    const user = userExtractor?.(req);
    const enabled = await client.isEnabled(flag, user);

    if (!enabled) {
      res.status(404).json({ error: 'Not found' });
      return;
    }

    next();
  };
}
