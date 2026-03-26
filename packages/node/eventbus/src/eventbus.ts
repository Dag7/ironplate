import axios, { type AxiosInstance } from 'axios';
import { createLogger } from '@ironplate/logger';
import type {
  EventEnvelope,
  EventHandler,
  PublishOptions,
  SubscribeOptions,
  EventBusOptions,
} from './types';

const DEFAULT_DAPR_HTTP_PORT = 3500;
const DEFAULT_PUBSUB_NAME = 'pubsub';
const DEFAULT_MAX_RETRIES = 3;
const DEFAULT_BASE_DELAY_MS = 500;

/**
 * Typed event bus client wrapping Dapr's pub/sub API.
 *
 * Publishes events through the Dapr sidecar HTTP API and provides
 * helpers for registering subscription handlers.
 */
export class EventBus {
  private readonly client: AxiosInstance;
  private readonly pubsubName: string;
  private readonly logger;
  private readonly maxRetries: number;
  private readonly baseDelayMs: number;
  private readonly handlers = new Map<string, EventHandler<unknown>[]>();

  constructor(options: EventBusOptions = {}) {
    const daprPort = options.daprPort ?? Number(process.env.DAPR_HTTP_PORT) ?? DEFAULT_DAPR_HTTP_PORT;
    const baseURL = options.daprUrl ?? `http://localhost:${daprPort}/v1.0`;

    this.pubsubName = options.pubsubName ?? DEFAULT_PUBSUB_NAME;
    this.maxRetries = options.maxRetries ?? DEFAULT_MAX_RETRIES;
    this.baseDelayMs = options.baseDelayMs ?? DEFAULT_BASE_DELAY_MS;
    this.logger = options.logger ?? createLogger('eventbus');

    this.client = axios.create({
      baseURL,
      timeout: options.timeout ?? 10_000,
      headers: { 'Content-Type': 'application/json' },
    });
  }

  /**
   * Publish an event to a topic via the Dapr sidecar.
   *
   * @param topic - The topic name to publish to.
   * @param data - The event payload.
   * @param options - Optional publish configuration.
   * @returns The created event envelope.
   */
  async publish<T>(topic: string, data: T, options?: PublishOptions): Promise<EventEnvelope<T>> {
    const envelope: EventEnvelope<T> = {
      id: options?.id ?? crypto.randomUUID(),
      source: options?.source ?? process.env.SERVICE_NAME ?? 'unknown',
      type: options?.type ?? topic,
      data,
      timestamp: new Date().toISOString(),
      correlationId: options?.correlationId,
    };

    const pubsubName = options?.pubsubName ?? this.pubsubName;
    const url = `/publish/${pubsubName}/${topic}`;

    await this.withRetry(async () => {
      this.logger.debug({ topic, eventId: envelope.id }, 'Publishing event');
      await this.client.post(url, envelope, {
        headers: options?.metadata
          ? { 'rawPayload': 'true', ...options.metadata }
          : undefined,
      });
      this.logger.info({ topic, eventId: envelope.id }, 'Event published');
    });

    return envelope;
  }

  /**
   * Register a handler for a topic. The handler will be invoked when a
   * matching event is received through the subscription middleware.
   *
   * @param topic - The topic to subscribe to.
   * @param handler - The event handler callback.
   * @param options - Optional subscription configuration.
   */
  subscribe<T = unknown>(
    topic: string,
    handler: EventHandler<T>,
    _options?: SubscribeOptions,
  ): void {
    const existing = this.handlers.get(topic) ?? [];
    existing.push(handler as EventHandler<unknown>);
    this.handlers.set(topic, existing);
    this.logger.info({ topic }, 'Subscription handler registered');
  }

  /**
   * Returns the registered handlers for a given topic.
   */
  getHandlers(topic: string): EventHandler<unknown>[] {
    return this.handlers.get(topic) ?? [];
  }

  /**
   * Returns all registered topic names.
   */
  getTopics(): string[] {
    return Array.from(this.handlers.keys());
  }

  /**
   * Returns the configured pubsub component name.
   */
  getPubsubName(): string {
    return this.pubsubName;
  }

  /**
   * Executes a function with exponential backoff retry.
   */
  private async withRetry(fn: () => Promise<void>): Promise<void> {
    let lastError: Error | undefined;

    for (let attempt = 0; attempt <= this.maxRetries; attempt++) {
      try {
        await fn();
        return;
      } catch (error) {
        lastError = error instanceof Error ? error : new Error(String(error));
        if (attempt < this.maxRetries) {
          const delay = this.baseDelayMs * Math.pow(2, attempt);
          this.logger.warn(
            { attempt: attempt + 1, maxRetries: this.maxRetries, delay, error: lastError.message },
            'Retrying after failure',
          );
          await new Promise((resolve) => setTimeout(resolve, delay));
        }
      }
    }

    this.logger.error({ error: lastError?.message }, 'All retry attempts exhausted');
    throw lastError;
  }
}
