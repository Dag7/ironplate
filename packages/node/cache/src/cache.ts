import Redis from 'ioredis';
import { createLogger } from '@ironplate/logger';
import type { CacheOptions, CacheClient } from './types';

const DEFAULT_TTL = 3600;

/**
 * Creates a Redis-backed cache client with serialization,
 * key prefixing, and TTL management.
 */
export function createCacheClient(options: CacheOptions = {}): CacheClient {
  const {
    url = process.env.REDIS_URL ?? 'redis://localhost:6379',
    prefix = '',
    defaultTtl = DEFAULT_TTL,
  } = options;

  const logger = options.logger ?? createLogger('cache');
  const redis = new Redis(url, { lazyConnect: true });

  redis.on('connect', () => logger.info('Cache connected'));
  redis.on('error', (err) => logger.error({ err }, 'Cache connection error'));

  const prefixKey = (key: string) => (prefix ? `${prefix}:${key}` : key);

  return {
    async get<T = unknown>(key: string): Promise<T | null> {
      const raw = await redis.get(prefixKey(key));
      if (raw === null) return null;
      try {
        return JSON.parse(raw) as T;
      } catch {
        return raw as unknown as T;
      }
    },

    async set<T = unknown>(key: string, value: T, ttl?: number): Promise<void> {
      const serialized = typeof value === 'string' ? value : JSON.stringify(value);
      const seconds = ttl ?? defaultTtl;
      await redis.set(prefixKey(key), serialized, 'EX', seconds);
    },

    async del(key: string): Promise<void> {
      await redis.del(prefixKey(key));
    },

    async exists(key: string): Promise<boolean> {
      const result = await redis.exists(prefixKey(key));
      return result === 1;
    },

    async flush(pattern?: string): Promise<number> {
      const searchPattern = prefixKey(pattern ?? '*');
      const keys = await redis.keys(searchPattern);
      if (keys.length === 0) return 0;
      const deleted = await redis.del(...keys);
      logger.info({ pattern: searchPattern, deleted }, 'Cache flushed');
      return deleted;
    },

    async disconnect(): Promise<void> {
      await redis.quit();
      logger.info('Cache disconnected');
    },
  };
}
