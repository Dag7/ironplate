import type { IJobStore, JobRecord } from './types';

/**
 * In-memory job store for development and testing.
 * NOT suitable for production (no persistence, no multi-process support).
 */
export class MemoryJobStore implements IJobStore {
  private jobs = new Map<string, JobRecord>();

  async enqueue(job: JobRecord): Promise<void> {
    this.jobs.set(job.id, { ...job });
  }

  async dequeue(jobNames: string[]): Promise<JobRecord | null> {
    const nameSet = new Set(jobNames);
    const now = new Date();

    // Find the highest priority pending job
    let best: JobRecord | null = null;
    for (const job of this.jobs.values()) {
      if (job.status !== 'pending' && job.status !== 'retrying') continue;
      if (!nameSet.has(job.name)) continue;
      if (job.scheduledFor && job.scheduledFor > now) continue;

      if (!best || job.priority < best.priority || (job.priority === best.priority && job.createdAt < best.createdAt)) {
        best = job;
      }
    }

    if (best) {
      best.status = 'running';
      best.startedAt = now;
      this.jobs.set(best.id, best);
    }

    return best ? { ...best } : null;
  }

  async update(id: string, updates: Partial<JobRecord>): Promise<void> {
    const job = this.jobs.get(id);
    if (job) {
      Object.assign(job, updates);
      this.jobs.set(id, job);
    }
  }

  async get(id: string): Promise<JobRecord | null> {
    const job = this.jobs.get(id);
    return job ? { ...job } : null;
  }

  /** Get all jobs (for testing) */
  getAll(): JobRecord[] {
    return Array.from(this.jobs.values()).map((j) => ({ ...j }));
  }

  /** Clear all jobs (for testing) */
  clear(): void {
    this.jobs.clear();
  }
}
