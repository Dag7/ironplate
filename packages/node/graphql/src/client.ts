import { GraphQLClient as GQLClient } from 'graphql-request';
import { createLogger } from '@ironplate/logger';
import type { GraphQLClientOptions, QueryOptions, Variables } from './types';

/**
 * Creates a pre-configured GraphQL client with Hasura admin auth,
 * structured logging, and error normalization.
 */
export function createGraphQLClient(options: GraphQLClientOptions) {
  const logger = options.logger ?? createLogger('graphql');
  const timeout = options.timeout ?? 30_000;

  const defaultHeaders: Record<string, string> = {
    'Content-Type': 'application/json',
    ...(options.adminSecret
      ? { 'x-hasura-admin-secret': options.adminSecret }
      : {}),
    ...(options.headers ?? {}),
  };

  const client = new GQLClient(options.endpoint, {
    headers: defaultHeaders,
    fetch: (input: any, init?: any) =>
      globalThis.fetch(input, { ...init, signal: AbortSignal.timeout(timeout) }),
  });

  return {
    /**
     * Execute a GraphQL query.
     */
    async query<T = unknown>(
      document: string,
      variables?: Variables,
      queryOptions?: QueryOptions,
    ): Promise<T> {
      const opName = queryOptions?.operationName ?? 'anonymous';
      logger.debug({ operation: opName }, 'Executing GraphQL query');

      try {
        const headers = queryOptions?.headers
          ? { ...defaultHeaders, ...queryOptions.headers }
          : undefined;

        const data = await client.request<T>({
          document,
          variables,
          requestHeaders: headers,
        });

        logger.debug({ operation: opName }, 'GraphQL query succeeded');
        return data;
      } catch (error) {
        logger.error(
          { operation: opName, error: error instanceof Error ? error.message : String(error) },
          'GraphQL query failed',
        );
        throw error;
      }
    },

    /**
     * Execute a GraphQL mutation.
     */
    async mutate<T = unknown>(
      document: string,
      variables?: Variables,
      queryOptions?: QueryOptions,
    ): Promise<T> {
      const opName = queryOptions?.operationName ?? 'anonymous';
      logger.debug({ operation: opName }, 'Executing GraphQL mutation');

      try {
        const headers = queryOptions?.headers
          ? { ...defaultHeaders, ...queryOptions.headers }
          : undefined;

        const data = await client.request<T>({
          document,
          variables,
          requestHeaders: headers,
        });

        logger.debug({ operation: opName }, 'GraphQL mutation succeeded');
        return data;
      } catch (error) {
        logger.error(
          { operation: opName, error: error instanceof Error ? error.message : String(error) },
          'GraphQL mutation failed',
        );
        throw error;
      }
    },

    /** Access the underlying graphql-request client */
    raw: client,
  };
}
