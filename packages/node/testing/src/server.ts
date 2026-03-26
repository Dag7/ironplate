import request from 'supertest';
import type { Express } from 'express';

/**
 * Creates a supertest agent from an Express app for integration testing.
 * No actual server is started — requests are handled in-process.
 */
export function createTestClient(app: Express) {
  return request(app);
}
