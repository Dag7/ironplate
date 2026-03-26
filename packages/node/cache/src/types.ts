import type { Logger } from '@ironplate/logger';

export interface CacheOptions {
  /** Redis connection URL (default: redis://localhost:6379) */
  url?: string;
  /** Key prefix for namespacing (default: '') */
  prefix?: string;
  /** Default TTL in seconds (default: 3600) */
  defaultTtl?: number;
  /** Logger instance */
  logger?: Logger;
}

export interface CacheClient {
  get<T = unknown>(key: string): Promise<T | null>;
  set<T = unknown>(key: string, value: T, ttl?: number): Promise<void>;
  del(key: string): Promise<void>;
  exists(key: string): Promise<boolean>;
  flush(pattern?: string): Promise<number>;
  disconnect(): Promise<void>;
}
