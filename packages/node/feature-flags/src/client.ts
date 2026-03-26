import { createLogger } from '@ironplate/logger';
import type { IFeatureFlagProvider, FeatureFlagUser, FeatureFlagConfig } from './types';

/**
 * Feature flag client that wraps a provider with logging and error handling.
 */
export class FeatureFlagClient {
  private readonly provider: IFeatureFlagProvider;
  private readonly logger;
  private initialized = false;

  constructor(config: FeatureFlagConfig) {
    this.provider = config.provider;
    this.logger = createLogger('feature-flags');
  }

  async initialize(): Promise<void> {
    if (this.initialized) return;
    await this.provider.initialize();
    this.initialized = true;
    this.logger.info('Feature flags initialized');
  }

  /**
   * Check if a feature flag is enabled.
   * Returns false on errors (fail-closed).
   */
  async isEnabled(flag: string, user?: FeatureFlagUser): Promise<boolean> {
    try {
      return await this.provider.isEnabled(flag, user);
    } catch (error) {
      this.logger.error({ flag, error }, 'Feature flag check failed');
      return false;
    }
  }

  /**
   * Get a dynamic config value.
   * Returns defaultValue on errors.
   */
  async getValue<T = unknown>(flag: string, user?: FeatureFlagUser, defaultValue?: T): Promise<T> {
    try {
      return await this.provider.getValue(flag, user, defaultValue);
    } catch (error) {
      this.logger.error({ flag, error }, 'Feature flag value fetch failed');
      return defaultValue as T;
    }
  }

  async shutdown(): Promise<void> {
    await this.provider.shutdown();
    this.logger.info('Feature flags shut down');
  }
}
