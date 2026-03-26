import * as jose from 'jose';
import type { JwtConfig, TokenPayload, ServiceTokenConfig } from './types';

/**
 * Creates and verifies JWT tokens using the jose library.
 */
export class TokenService {
  private readonly secret: Uint8Array;
  private readonly issuer?: string;
  private readonly audience?: string;
  private readonly accessExpiry: string;
  private readonly refreshExpiry: string;

  constructor(config: JwtConfig) {
    this.secret = new TextEncoder().encode(config.secret);
    this.issuer = config.issuer;
    this.audience = config.audience;
    this.accessExpiry = config.accessTokenExpiry ?? '15m';
    this.refreshExpiry = config.refreshTokenExpiry ?? '7d';
  }

  /**
   * Sign an access token.
   */
  async signAccessToken(payload: Record<string, unknown>): Promise<string> {
    const builder = new jose.SignJWT(payload)
      .setProtectedHeader({ alg: 'HS256' })
      .setIssuedAt()
      .setExpirationTime(this.accessExpiry);

    if (this.issuer) builder.setIssuer(this.issuer);
    if (this.audience) builder.setAudience(this.audience);

    return builder.sign(this.secret);
  }

  /**
   * Sign a refresh token.
   */
  async signRefreshToken(payload: Record<string, unknown>): Promise<string> {
    const builder = new jose.SignJWT({ ...payload, type: 'refresh' })
      .setProtectedHeader({ alg: 'HS256' })
      .setIssuedAt()
      .setExpirationTime(this.refreshExpiry)
      .setJti(crypto.randomUUID());

    if (this.issuer) builder.setIssuer(this.issuer);
    if (this.audience) builder.setAudience(this.audience);

    return builder.sign(this.secret);
  }

  /**
   * Verify and decode a token.
   */
  async verify(token: string): Promise<TokenPayload> {
    const options: jose.JWTVerifyOptions = {};
    if (this.issuer) options.issuer = this.issuer;
    if (this.audience) options.audience = this.audience;

    const { payload } = await jose.jwtVerify(token, this.secret, options);
    return payload as TokenPayload;
  }

  /**
   * Decode a token without verification (for inspection only).
   */
  decode(token: string): TokenPayload | null {
    try {
      const decoded = jose.decodeJwt(token);
      return decoded as TokenPayload;
    } catch {
      return null;
    }
  }
}

/**
 * Creates a service-to-service token provider for internal API calls.
 */
export class ServiceTokenProvider {
  private readonly tokenService: TokenService;
  private readonly serviceName: string;
  private cachedToken?: string;
  private tokenExpiry = 0;

  constructor(config: ServiceTokenConfig) {
    this.serviceName = config.serviceName;
    this.tokenService = new TokenService({
      secret: config.secret,
      issuer: config.serviceName,
      accessTokenExpiry: config.expiry ?? '1h',
    });
  }

  /**
   * Get a valid service token, refreshing if needed.
   */
  async getToken(): Promise<string> {
    const now = Math.floor(Date.now() / 1000);
    // Refresh 5 minutes before expiry
    if (this.cachedToken && this.tokenExpiry - now > 300) {
      return this.cachedToken;
    }

    this.cachedToken = await this.tokenService.signAccessToken({
      sub: `service:${this.serviceName}`,
      type: 'service',
      serviceName: this.serviceName,
    });

    const decoded = this.tokenService.decode(this.cachedToken);
    this.tokenExpiry = decoded?.exp ?? now + 3600;

    return this.cachedToken;
  }

  /**
   * Get an authorization header value.
   */
  async getAuthHeader(): Promise<string> {
    const token = await this.getToken();
    return `Bearer ${token}`;
  }
}
