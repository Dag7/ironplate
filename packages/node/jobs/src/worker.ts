import { createLogger } from '@ironplate/logger';
import type { Logger } from '@ironplate/logger';
import type { JobDefinition, JobContext, JobRecord, IJobStore, JobOptions } from './types';

const DEFAULT_POLL_INTERVAL_MS = 1000;
const DEFAULT_MAX_RETRIES = 3;
const DEFAULT_RETRY_DELAY_MS = 1000;
const DEFAULT_TIMEOUT_MS = 30_000;

/**
 * Job worker that processes jobs from a store.
 *
 * @example
 * ```ts
 * const worker = new JobWorker(store);
 *
 * worker.register({
 *   name: 'send-email',
 *   handler: async (payload, ctx) => {
 *     ctx.logger.info({ to: payload.to }, 'Sending email');
 *     await emailService.send(payload);
 *   },
 * });
 *
 * // Enqueue a job
 * await worker.enqueue('send-email', { to: 'user@example.com', subject: 'Hello' });
 *
 * // Start processing
 * worker.start();
 * ```
 */
export class JobWorker {
  private readonly store: IJobStore;
  private readonly logger: Logger;
  private readonly definitions = new Map<string, JobDefinition>();
  private pollInterval: number;
  private timer?: ReturnType<typeof setInterval>;
  private running = false;

  constructor(store: IJobStore, options?: { logger?: Logger; pollIntervalMs?: number }) {
    this.store = store;
    this.logger = options?.logger ?? createLogger('jobs');
    this.pollInterval = options?.pollIntervalMs ?? DEFAULT_POLL_INTERVAL_MS;
  }

  /**
   * Register a job definition.
   */
  register<TPayload = unknown, TResult = unknown>(
    definition: JobDefinition<TPayload, TResult>,
  ): this {
    this.definitions.set(definition.name, definition as JobDefinition);
    this.logger.debug({ job: definition.name }, 'Job registered');
    return this;
  }

  /**
   * Enqueue a job for processing.
   */
  async enqueue<TPayload = unknown>(
    name: string,
    payload: TPayload,
    options?: JobOptions,
  ): Promise<string> {
    const definition = this.definitions.get(name);
    if (!definition) {
      throw new Error(`Unknown job type: ${name}`);
    }

    const id = options?.id ?? crypto.randomUUID();
    const record: JobRecord<TPayload> = {
      id,
      name,
      payload,
      status: 'pending',
      attempt: 0,
      maxRetries: definition.maxRetries ?? DEFAULT_MAX_RETRIES,
      createdAt: new Date(),
      scheduledFor: options?.delayMs ? new Date(Date.now() + options.delayMs) : undefined,
      priority: options?.priority ?? 0,
    };

    await this.store.enqueue(record as JobRecord);
    this.logger.info({ jobId: id, name }, 'Job enqueued');
    return id;
  }

  /**
   * Start the worker polling loop.
   */
  start(): void {
    if (this.running) return;
    this.running = true;
    this.logger.info({ jobs: Array.from(this.definitions.keys()) }, 'Job worker started');

    this.timer = setInterval(() => {
      this.poll().catch((err) => {
        this.logger.error({ err }, 'Poll error');
      });
    }, this.pollInterval);
  }

  /**
   * Stop the worker.
   */
  stop(): void {
    this.running = false;
    if (this.timer) {
      clearInterval(this.timer);
      this.timer = undefined;
    }
    this.logger.info('Job worker stopped');
  }

  private async poll(): Promise<void> {
    const jobNames = Array.from(this.definitions.keys());
    const record = await this.store.dequeue(jobNames);
    if (!record) return;

    const definition = this.definitions.get(record.name);
    if (!definition) return;

    const attempt = record.attempt + 1;
    const jobLogger = createLogger(`jobs:${record.name}:${record.id}`);

    const timeoutMs = definition.timeoutMs ?? DEFAULT_TIMEOUT_MS;
    const controller = new AbortController();
    const timeout = setTimeout(() => controller.abort(), timeoutMs);

    const ctx: JobContext = {
      jobId: record.id,
      attempt,
      logger: jobLogger,
      signal: controller.signal,
    };

    try {
      jobLogger.info('Processing job');
      const result = await definition.handler(record.payload, ctx);

      await this.store.update(record.id, {
        status: 'completed',
        result,
        attempt,
        completedAt: new Date(),
      });

      jobLogger.info('Job completed');
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : String(error);
      jobLogger.error({ error: errorMessage }, 'Job failed');

      const maxRetries = definition.maxRetries ?? DEFAULT_MAX_RETRIES;
      if (attempt < maxRetries) {
        const delay = (definition.retryDelayMs ?? DEFAULT_RETRY_DELAY_MS) * Math.pow(2, attempt - 1);
        await this.store.update(record.id, {
          status: 'retrying',
          attempt,
          error: errorMessage,
          scheduledFor: new Date(Date.now() + delay),
        });
        jobLogger.info({ nextAttempt: attempt + 1, delayMs: delay }, 'Job scheduled for retry');
      } else {
        await this.store.update(record.id, {
          status: 'failed',
          attempt,
          error: errorMessage,
          completedAt: new Date(),
        });
        jobLogger.error({ attempts: attempt }, 'Job exhausted all retries');
      }
    } finally {
      clearTimeout(timeout);
    }
  }
}
