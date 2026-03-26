import type { Express, Router, Request, Response, NextFunction } from 'express';
import type { Logger } from '@ironplate/logger';
import type { Server } from 'http';

export type HttpMethod = 'get' | 'post' | 'put' | 'patch' | 'delete' | 'all';

export interface ServiceConfig {
  /** Service name, used in logs and health checks */
  name: string;
  /** HTTP port (default: 3000 or PORT env var) */
  port?: number;
  /** Base path prefix for all routes (e.g., '/api/v1') */
  basePath?: string;
  /** Service version */
  version?: string;
  /** Logger instance (auto-created if not provided) */
  logger?: Logger;
  /** Enable CORS (default: true) */
  cors?: boolean | object;
  /** Enable Helmet security headers (default: true) */
  helmet?: boolean | object;
  /** Enable compression (default: true) */
  compression?: boolean;
  /** Enable JSON body parsing (default: true) */
  jsonBody?: boolean | { limit?: string };
}

export interface IPlugin {
  /** Unique plugin name */
  readonly name: string;
  /** Execution order (lower = earlier). Built-in: 1-10, User: 50+ */
  readonly order: number;
  /** Names of plugins this one depends on */
  readonly dependencies?: string[];
  /** Called during service build to register middleware/routes */
  initialize(ctx: PluginContext): Promise<void>;
  /** Called after all plugins are initialized */
  finalize?(ctx: PluginContext): Promise<void>;
  /** Called during service shutdown for cleanup */
  dispose?(): Promise<void>;
}

export interface MiddlewareConfig {
  /** Middleware function */
  handler: (req: Request, res: Response, next: NextFunction) => void;
  /** Optional path to mount on */
  path?: string;
  /** Execution order (lower = earlier) */
  order?: number;
}

export interface RouteConfig {
  /** Route path */
  path: string;
  /** Express Router */
  router: Router;
  /** Skip auth for these routes */
  skipAuth?: boolean;
}

export interface PluginContext {
  /** Service configuration */
  readonly config: Required<Pick<ServiceConfig, 'name' | 'port' | 'version'>> & ServiceConfig;
  /** Logger instance */
  readonly logger: Logger;
  /** The Express app instance */
  readonly app: Express;
  /** Register middleware */
  addMiddleware(config: MiddlewareConfig): void;
  /** Register routes */
  addRoutes(config: RouteConfig): void;
  /** Paths excluded from auth */
  addAuthExcludePath(path: string): void;
  getAuthExcludePaths(): string[];
  /** Cross-plugin service registry */
  registerService<T>(name: string, instance: T): void;
  getService<T>(name: string): T | undefined;
  hasService(name: string): boolean;
  /** Get a registered plugin by name */
  getPlugin<T extends IPlugin>(name: string): T | undefined;
}

export interface ServiceInstance {
  /** The Express app */
  readonly app: Express;
  /** Start listening for requests */
  start(): Promise<Server>;
  /** Graceful shutdown */
  stop(): Promise<void>;
  /** Get a registered service from the plugin context */
  getService<T>(name: string): T | undefined;
}
