import type { Logger } from '@ironplate/logger';

export interface GraphQLClientOptions {
  /** Hasura/GraphQL endpoint URL */
  endpoint: string;
  /** Admin secret for Hasura */
  adminSecret?: string;
  /** Additional default headers */
  headers?: Record<string, string>;
  /** Request timeout in ms (default: 30000) */
  timeout?: number;
  /** Logger instance */
  logger?: Logger;
}

export interface QueryOptions {
  /** Override headers for this request */
  headers?: Record<string, string>;
  /** Operation name for logging */
  operationName?: string;
}

export type Variables = Record<string, unknown>;
