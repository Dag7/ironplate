import type { IFeatureFlagProvider, FeatureFlagUser } from './types';

/**
 * In-memory feature flag provider for local development and testing.
 *
 * @example
 * ```ts
 * const provider = new LocalProvider({
 *   'new-dashboard': true,
 *   'max-uploads': 10,
 *   'beta-features': false,
 * });
 * ```
 */
export class LocalProvider implements IFeatureFlagProvider {
  private flags: Map<string, unknown>;

  constructor(flags: Record<string, unknown> = {}) {
    this.flags = new Map(Object.entries(flags));
  }

  async initialize(): Promise<void> {
    // No-op for local provider
  }

  async isEnabled(flag: string, _user?: FeatureFlagUser): Promise<boolean> {
    const value = this.flags.get(flag);
    return value === true;
  }

  async getValue<T = unknown>(flag: string, _user?: FeatureFlagUser, defaultValue?: T): Promise<T> {
    const value = this.flags.get(flag);
    return (value !== undefined ? value : defaultValue) as T;
  }

  async shutdown(): Promise<void> {
    // No-op
  }

  /** Set a flag value at runtime (useful in tests) */
  set(flag: string, value: unknown): void {
    this.flags.set(flag, value);
  }

  /** Remove a flag */
  remove(flag: string): void {
    this.flags.delete(flag);
  }

  /** Clear all flags */
  clear(): void {
    this.flags.clear();
  }
}
