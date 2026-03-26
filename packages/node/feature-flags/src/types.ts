export interface FeatureFlagUser {
  userId: string;
  email?: string;
  [key: string]: unknown;
}

export interface IFeatureFlagProvider {
  /** Initialize the provider (connect, sync, etc.) */
  initialize(): Promise<void>;
  /** Check if a feature is enabled for a user */
  isEnabled(flag: string, user?: FeatureFlagUser): Promise<boolean>;
  /** Get a dynamic config value for a user */
  getValue<T = unknown>(flag: string, user?: FeatureFlagUser, defaultValue?: T): Promise<T>;
  /** Shutdown the provider */
  shutdown(): Promise<void>;
}

export interface FeatureFlagConfig {
  /** The provider implementation */
  provider: IFeatureFlagProvider;
}
