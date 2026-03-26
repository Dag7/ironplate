import type { Logger } from '@ironplate/logger';

export interface JobDefinition<TPayload = unknown, TResult = unknown> {
  /** Unique job type name */
  name: string;
  /** Handler function that processes the job */
  handler: (payload: TPayload, ctx: JobContext) => Promise<TResult>;
  /** Maximum number of retry attempts (default: 3) */
  maxRetries?: number;
  /** Base delay between retries in ms (default: 1000, exponential backoff) */
  retryDelayMs?: number;
  /** Job timeout in ms (default: 30000) */
  timeoutMs?: number;
}

export interface JobContext {
  /** Job ID */
  jobId: string;
  /** Current attempt number (starts at 1) */
  attempt: number;
  /** Logger scoped to this job */
  logger: Logger;
  /** Signal that aborts if the job times out */
  signal: AbortSignal;
}

export interface JobOptions {
  /** Custom job ID (auto-generated if not provided) */
  id?: string;
  /** Delay before processing in ms */
  delayMs?: number;
  /** Priority (lower = higher priority, default: 0) */
  priority?: number;
}

export interface JobRecord<TPayload = unknown> {
  id: string;
  name: string;
  payload: TPayload;
  status: 'pending' | 'running' | 'completed' | 'failed' | 'retrying';
  attempt: number;
  maxRetries: number;
  result?: unknown;
  error?: string;
  createdAt: Date;
  startedAt?: Date;
  completedAt?: Date;
  scheduledFor?: Date;
  priority: number;
}

export interface IJobStore {
  /** Enqueue a job */
  enqueue(job: JobRecord): Promise<void>;
  /** Dequeue the next available job */
  dequeue(jobNames: string[]): Promise<JobRecord | null>;
  /** Update job status */
  update(id: string, updates: Partial<JobRecord>): Promise<void>;
  /** Get a job by ID */
  get(id: string): Promise<JobRecord | null>;
}
