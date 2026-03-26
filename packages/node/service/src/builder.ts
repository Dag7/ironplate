import express from 'express';
import cors from 'cors';
import helmet from 'helmet';
import compression from 'compression';
import { createLogger } from '@ironplate/logger';
import { createHealthRouter } from '@ironplate/health';
import type { Router } from 'express';
import type { Logger } from '@ironplate/logger';
import type { Server } from 'http';
import type { ServiceConfig, IPlugin, ServiceInstance, RouteConfig } from './types';
import { ServicePluginContext } from './context';

/**
 * Fluent builder for composing microservices with plugins.
 *
 * @example
 * ```ts
 * const service = await createService('my-api')
 *   .port(3001)
 *   .use(authPlugin())
 *   .use(metricsPlugin())
 *   .routes('/users', usersRouter)
 *   .routes('/health', healthRouter, { skipAuth: true })
 *   .build();
 *
 * await service.start();
 * ```
 */
export class ServiceBuilder {
  private config: ServiceConfig;
  private plugins: IPlugin[] = [];
  private routeConfigs: RouteConfig[] = [];
  private customLogger?: Logger;

  constructor(name: string) {
    this.config = { name };
  }

  /** Set the HTTP port */
  port(port: number): this {
    this.config.port = port;
    return this;
  }

  /** Set a base path prefix for all routes */
  basePath(path: string): this {
    this.config.basePath = path;
    return this;
  }

  /** Set the service version */
  version(version: string): this {
    this.config.version = version;
    return this;
  }

  /** Provide a custom logger */
  logger(logger: Logger): this {
    this.customLogger = logger;
    return this;
  }

  /** Configure CORS (true to enable with defaults, object for custom config) */
  withCors(options?: object | boolean): this {
    this.config.cors = options ?? true;
    return this;
  }

  /** Configure Helmet security headers */
  withHelmet(options?: object | boolean): this {
    this.config.helmet = options ?? true;
    return this;
  }

  /** Enable/disable compression */
  withCompression(enabled = true): this {
    this.config.compression = enabled;
    return this;
  }

  /** Register a plugin */
  use(plugin: IPlugin): this {
    this.plugins.push(plugin);
    return this;
  }

  /** Add route handlers */
  routes(path: string, router: Router, options?: { skipAuth?: boolean }): this {
    this.routeConfigs.push({
      path,
      router,
      skipAuth: options?.skipAuth,
    });
    return this;
  }

  /** Build and return the service instance */
  async build(): Promise<ServiceInstance> {
    const logger = this.customLogger ?? createLogger(this.config.name);
    const port = this.config.port ?? parseInt(process.env.PORT ?? '3000', 10);
    const version = this.config.version ?? process.env.SERVICE_VERSION ?? '0.0.0';

    const resolvedConfig = {
      ...this.config,
      port,
      version,
      cors: this.config.cors ?? true,
      helmet: this.config.helmet ?? true,
      compression: this.config.compression ?? true,
      jsonBody: this.config.jsonBody ?? true,
    };

    const app = express();
    const ctx = new ServicePluginContext(resolvedConfig, logger, app);

    // Register built-in middleware
    if (resolvedConfig.helmet) {
      const helmetOpts = typeof resolvedConfig.helmet === 'object' ? resolvedConfig.helmet : {};
      ctx.addMiddleware({ handler: helmet(helmetOpts), order: 1 });
    }
    if (resolvedConfig.cors) {
      const corsOpts = typeof resolvedConfig.cors === 'object' ? resolvedConfig.cors : {};
      ctx.addMiddleware({ handler: cors(corsOpts), order: 2 });
    }
    if (resolvedConfig.compression) {
      ctx.addMiddleware({ handler: compression(), order: 3 });
    }
    if (resolvedConfig.jsonBody) {
      const limit = typeof resolvedConfig.jsonBody === 'object'
        ? resolvedConfig.jsonBody.limit
        : '1mb';
      ctx.addMiddleware({ handler: express.json({ limit }), order: 4 });
    }

    // Built-in health routes (always present, no auth)
    ctx.addRoutes({
      path: '/',
      router: createHealthRouter({ serviceName: resolvedConfig.name }),
      skipAuth: true,
    });

    // Sort plugins by order, validate dependencies
    const sorted = this.sortPlugins(this.plugins);

    // Initialize plugins
    for (const plugin of sorted) {
      ctx.registerPlugin(plugin);
      logger.debug({ plugin: plugin.name }, 'Initializing plugin');
      await plugin.initialize(ctx);
    }

    // Finalize plugins
    for (const plugin of sorted) {
      if (plugin.finalize) {
        await plugin.finalize(ctx);
      }
    }

    // Register user routes
    for (const route of this.routeConfigs) {
      ctx.addRoutes(route);
    }

    // Apply all middleware and routes to the Express app
    ctx.apply();

    // Build the service instance
    let server: Server | undefined;

    const instance: ServiceInstance = {
      app,

      async start(): Promise<Server> {
        return new Promise((resolve) => {
          server = app.listen(port, () => {
            logger.info({ port, service: resolvedConfig.name, version }, 'Service started');
            resolve(server!);
          });
        });
      },

      async stop(): Promise<void> {
        logger.info('Shutting down service...');

        // Dispose plugins in reverse order
        for (const plugin of [...sorted].reverse()) {
          if (plugin.dispose) {
            try {
              await plugin.dispose();
            } catch (err) {
              logger.error({ plugin: plugin.name, err }, 'Plugin dispose error');
            }
          }
        }

        // Close HTTP server
        if (server) {
          await new Promise<void>((resolve, reject) => {
            server!.close((err) => (err ? reject(err) : resolve()));
          });
        }

        logger.info('Service stopped');
      },

      getService<T>(name: string): T | undefined {
        return ctx.getService<T>(name);
      },
    };

    // Register graceful shutdown handlers
    const shutdown = async () => {
      await instance.stop();
      process.exit(0);
    };
    process.on('SIGINT', shutdown);
    process.on('SIGTERM', shutdown);

    return instance;
  }

  /** Topological sort plugins by dependencies and order */
  private sortPlugins(plugins: IPlugin[]): IPlugin[] {
    const byName = new Map(plugins.map((p) => [p.name, p]));
    const visited = new Set<string>();
    const result: IPlugin[] = [];

    const visit = (plugin: IPlugin) => {
      if (visited.has(plugin.name)) return;
      visited.add(plugin.name);

      for (const dep of plugin.dependencies ?? []) {
        const depPlugin = byName.get(dep);
        if (depPlugin) {
          visit(depPlugin);
        }
      }
      result.push(plugin);
    };

    // Sort by order first, then resolve dependencies
    const sorted = [...plugins].sort((a, b) => a.order - b.order);
    for (const plugin of sorted) {
      visit(plugin);
    }

    return result;
  }
}

/**
 * Entry point for building a service.
 *
 * @param name - The service name
 * @returns A ServiceBuilder instance
 *
 * @example
 * ```ts
 * const service = await createService('user-api')
 *   .port(3001)
 *   .routes('/users', usersRouter)
 *   .build();
 *
 * await service.start();
 * ```
 */
export function createService(name: string): ServiceBuilder {
  return new ServiceBuilder(name);
}
