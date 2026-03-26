import type { Logger } from '@ironplate/logger';

/** Standard event envelope wrapping all published events. */
export interface EventEnvelope<T> {
  /** Unique event identifier. */
  id: string;
  /** The originating service name. */
  source: string;
  /** Event type, typically matching the topic name. */
  type: string;
  /** The event payload. */
  data: T;
  /** ISO 8601 timestamp of when the event was created. */
  timestamp: string;
  /** Distributed trace correlation ID. */
  correlationId?: string;
}

/** Callback invoked when a subscribed event is received. */
export type EventHandler<T> = (envelope: EventEnvelope<T>) => Promise<void>;

/** Options for publishing an event. */
export interface PublishOptions {
  /** Override the event ID (defaults to a random UUID). */
  id?: string;
  /** Override the source service name. */
  source?: string;
  /** Override the event type (defaults to the topic name). */
  type?: string;
  /** Correlation ID for distributed tracing. */
  correlationId?: string;
  /** Override the pubsub component name for this publish call. */
  pubsubName?: string;
  /** Additional Dapr metadata headers. */
  metadata?: Record<string, string>;
}

/** Options for subscribing to a topic. */
export interface SubscribeOptions {
  /** Route path for the subscription endpoint. Defaults to /events/<topic>. */
  route?: string;
  /** Dead letter topic for failed messages. */
  deadLetterTopic?: string;
}

/** Configuration options for the EventBus instance. */
export interface EventBusOptions {
  /** The Dapr sidecar HTTP port. Defaults to DAPR_HTTP_PORT env var or 3500. */
  daprPort?: number;
  /** Full Dapr sidecar base URL (overrides daprPort). */
  daprUrl?: string;
  /** The Dapr pubsub component name. Defaults to 'pubsub'. */
  pubsubName?: string;
  /** Request timeout in milliseconds. Defaults to 10000. */
  timeout?: number;
  /** Maximum number of retry attempts. Defaults to 3. */
  maxRetries?: number;
  /** Base delay in milliseconds for exponential backoff. Defaults to 500. */
  baseDelayMs?: number;
  /** Logger instance. Falls back to creating a default logger. */
  logger?: Logger;
}
