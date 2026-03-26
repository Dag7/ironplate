import type { Request, Response, NextFunction } from 'express';

export interface JwtConfig {
  /** Secret key for HS256 or JWKS URL for RS256 */
  secret: string;
  /** Token issuer */
  issuer?: string;
  /** Token audience */
  audience?: string;
  /** Access token expiry (default: '15m') */
  accessTokenExpiry?: string;
  /** Refresh token expiry (default: '7d') */
  refreshTokenExpiry?: string;
}

export interface TokenPayload {
  sub: string;
  iss?: string;
  aud?: string;
  exp?: number;
  iat?: number;
  jti?: string;
  [key: string]: unknown;
}

export interface ServiceTokenConfig {
  /** Service name (used as issuer) */
  serviceName: string;
  /** Shared secret for service-to-service auth */
  secret: string;
  /** Token expiry (default: '1h') */
  expiry?: string;
}

export interface AuthMiddlewareOptions {
  /** JWT configuration */
  jwt: JwtConfig;
  /** Paths to exclude from auth (e.g., ['/healthz', '/readyz']) */
  excludePaths?: string[];
  /** Custom token extractor (default: Bearer token from Authorization header) */
  tokenExtractor?: (req: Request) => string | undefined;
  /** Called when auth succeeds — use to attach user to request */
  onAuthenticated?: (req: Request, payload: TokenPayload) => void;
}

export interface AuthenticatedRequest extends Request {
  auth?: {
    payload: TokenPayload;
    userId: string;
    isServiceCall: boolean;
    serviceName?: string;
  };
}

export type AuthMiddleware = (req: Request, res: Response, next: NextFunction) => void;
