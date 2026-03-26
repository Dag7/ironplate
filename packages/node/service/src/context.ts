import type express from 'express';
import type { Logger } from '@ironplate/logger';
import type {
  ServiceConfig,
  IPlugin,
  PluginContext,
  MiddlewareConfig,
  RouteConfig,
} from './types';

export class ServicePluginContext implements PluginContext {
  readonly config: Required<Pick<ServiceConfig, 'name' | 'port' | 'version'>> & ServiceConfig;
  readonly logger: Logger;
  readonly app: express.Express;

  private middlewares: MiddlewareConfig[] = [];
  private routes: RouteConfig[] = [];
  private authExcludePaths: string[] = [];
  private services = new Map<string, unknown>();
  private plugins = new Map<string, IPlugin>();

  constructor(
    config: Required<Pick<ServiceConfig, 'name' | 'port' | 'version'>> & ServiceConfig,
    logger: Logger,
    app: express.Express,
  ) {
    this.config = config;
    this.logger = logger;
    this.app = app;
  }

  addMiddleware(config: MiddlewareConfig): void {
    this.middlewares.push(config);
  }

  addRoutes(config: RouteConfig): void {
    this.routes.push(config);
    if (config.skipAuth) {
      this.authExcludePaths.push(config.path);
    }
  }

  addAuthExcludePath(path: string): void {
    this.authExcludePaths.push(path);
  }

  getAuthExcludePaths(): string[] {
    return [...this.authExcludePaths];
  }

  registerService<T>(name: string, instance: T): void {
    this.services.set(name, instance);
  }

  getService<T>(name: string): T | undefined {
    return this.services.get(name) as T | undefined;
  }

  hasService(name: string): boolean {
    return this.services.has(name);
  }

  registerPlugin(plugin: IPlugin): void {
    this.plugins.set(plugin.name, plugin);
  }

  getPlugin<T extends IPlugin>(name: string): T | undefined {
    return this.plugins.get(name) as T | undefined;
  }

  /** Apply all collected middleware and routes to the Express app, sorted by order */
  apply(): void {
    // Sort and apply middleware
    const sorted = [...this.middlewares].sort((a, b) => (a.order ?? 50) - (b.order ?? 50));
    for (const mw of sorted) {
      if (mw.path) {
        this.app.use(mw.path, mw.handler);
      } else {
        this.app.use(mw.handler);
      }
    }

    // Apply routes under basePath
    const basePath = this.config.basePath ?? '';
    for (const route of this.routes) {
      const fullPath = basePath + route.path;
      this.app.use(fullPath, route.router);
    }
  }

  getRegisteredPlugins(): IPlugin[] {
    return Array.from(this.plugins.values());
  }
}
