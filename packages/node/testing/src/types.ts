import type { Express } from 'express';

export interface TestServerOptions {
  /** The Express app to test */
  app: Express;
}

export interface MockFactoryOptions<T> {
  /** Default values for the mock */
  defaults: T;
}
