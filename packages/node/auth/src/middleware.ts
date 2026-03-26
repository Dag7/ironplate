import { UnauthorizedError } from '@ironplate/errors';
import { createLogger } from '@ironplate/logger';
import { TokenService } from './jwt';
import type { Request, Response, NextFunction } from 'express';
import type { AuthMiddlewareOptions, AuthenticatedRequest, TokenPayload } from './types';

/**
 * Creates Express middleware that validates JWT tokens on incoming requests.
 *
 * @example
 * ```ts
 * app.use(createAuthMiddleware({
 *   jwt: { secret: process.env.JWT_SECRET! },
 *   excludePaths: ['/healthz', '/readyz', '/docs'],
 * }));
 * ```
 */
export function createAuthMiddleware(options: AuthMiddlewareOptions) {
  const logger = createLogger('auth');
  const tokenService = new TokenService(options.jwt);

  const excludePaths = new Set(options.excludePaths ?? ['/healthz', '/readyz']);

  const extractToken = options.tokenExtractor ?? ((req: Request): string | undefined => {
    const header = req.headers.authorization;
    if (header?.startsWith('Bearer ')) {
      return header.slice(7);
    }
    return undefined;
  });

  return async (req: Request, res: Response, next: NextFunction): Promise<void> => {
    // Skip excluded paths
    if (excludePaths.has(req.path)) {
      return next();
    }

    const token = extractToken(req);
    if (!token) {
      return next(new UnauthorizedError('Missing authentication token'));
    }

    try {
      const payload = await tokenService.verify(token);

      // Attach auth context to request
      const authReq = req as AuthenticatedRequest;
      const isServiceCall = payload.type === 'service';

      authReq.auth = {
        payload,
        userId: typeof payload.sub === 'string' ? payload.sub : '',
        isServiceCall,
        serviceName: isServiceCall ? (payload.serviceName as string) : undefined,
      };

      if (options.onAuthenticated) {
        options.onAuthenticated(req, payload);
      }

      next();
    } catch (error) {
      logger.debug({ error }, 'Token verification failed');
      next(new UnauthorizedError('Invalid or expired token'));
    }
  };
}

/**
 * Middleware that requires a specific claim in the JWT payload.
 */
export function requireClaim(claim: string, value?: unknown) {
  return (req: Request, _res: Response, next: NextFunction): void => {
    const authReq = req as AuthenticatedRequest;
    if (!authReq.auth?.payload) {
      return next(new UnauthorizedError('Not authenticated'));
    }

    const claimValue = authReq.auth.payload[claim];
    if (claimValue === undefined) {
      return next(new UnauthorizedError(`Missing required claim: ${claim}`));
    }

    if (value !== undefined && claimValue !== value) {
      return next(new UnauthorizedError(`Invalid claim value for: ${claim}`));
    }

    next();
  };
}

/**
 * Middleware that requires the request to be from a service (S2S auth).
 */
export function requireServiceAuth() {
  return (req: Request, _res: Response, next: NextFunction): void => {
    const authReq = req as AuthenticatedRequest;
    if (!authReq.auth?.isServiceCall) {
      return next(new UnauthorizedError('Service authentication required'));
    }
    next();
  };
}
