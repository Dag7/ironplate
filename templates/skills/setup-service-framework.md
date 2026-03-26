# Service Framework — Builder Pattern

## Arguments
- `service`: Service name to configure
- `plugins` (optional): Comma-separated plugins to add (e.g., `auth,metrics,tracing`)

## Architecture

```
createService(name)
  ├── Built-in: helmet, cors, compression, json, health
  ├── .use(plugin)  →  IPlugin { initialize → finalize → dispose }
  ├── .routes(path, router)
  └── .build()  →  ServiceInstance { start(), stop(), getService() }
```

## Plugin Execution Order

| Order | Plugin | Purpose |
|-------|--------|---------|
| 1 | helmet | Security headers (built-in) |
| 2 | cors | Cross-origin (built-in) |
| 3 | compression | Response compression (built-in) |
| 4 | json | Body parsing (built-in) |
| 5 | health | Health check routes (built-in) |
| 10 | metrics | Prometheus instrumentation |
| 20 | tracing | OpenTelemetry spans |
| 30 | auth | JWT authentication |
| 40 | validation | Request validation |
| 100 | error | Error handler (last) |

## Steps

### 1. Basic Service Setup

```typescript
import { createService } from '@ironplate/service';
import { usersRouter } from './routes/users';

const service = await createService('my-service')
  .port(3001)
  .basePath('/api/v1')
  .version('1.0.0')
  .routes('/users', usersRouter)
  .build();

await service.start();
```

### 2. Adding Plugins

```typescript
import { createService, type IPlugin, type PluginContext } from '@ironplate/service';
import { createAuthMiddleware } from '@ironplate/auth';
import { createMetricsRegistry, createHttpMetrics, createMetricsEndpoint } from '@ironplate/metrics';

// Auth plugin
const authPlugin: IPlugin = {
  name: 'auth',
  order: 30,
  async initialize(ctx: PluginContext) {
    const middleware = createAuthMiddleware({
      jwt: { secret: process.env.JWT_SECRET! },
      excludePaths: ctx.getAuthExcludePaths(),
    });
    ctx.addMiddleware({ handler: middleware, order: 30 });
  },
};

// Metrics plugin
const metricsPlugin: IPlugin = {
  name: 'metrics',
  order: 10,
  async initialize(ctx: PluginContext) {
    const registry = createMetricsRegistry({ serviceName: ctx.config.name });
    const { middleware } = createHttpMetrics(registry, { serviceName: ctx.config.name });
    ctx.addMiddleware({ handler: middleware, order: 10 });
    ctx.addRoutes({
      path: '/',
      router: createMetricsEndpoint(registry),
      skipAuth: true,
    });
    ctx.registerService('metricsRegistry', registry);
  },
};

const service = await createService('my-service')
  .use(metricsPlugin)
  .use(authPlugin)
  .routes('/users', usersRouter)
  .routes('/webhooks', webhookRouter, { skipAuth: true })
  .build();
```

### 3. Creating Custom Plugins

```typescript
import type { IPlugin, PluginContext } from '@ironplate/service';

export function rateLimitPlugin(maxRequests = 100): IPlugin {
  return {
    name: 'rate-limit',
    order: 25,
    dependencies: ['auth'], // runs after auth
    async initialize(ctx: PluginContext) {
      ctx.addMiddleware({
        handler: (req, res, next) => {
          // rate limiting logic
          next();
        },
        order: 25,
      });
      ctx.logger.info({ maxRequests }, 'Rate limiting enabled');
    },
    async dispose() {
      // cleanup if needed
    },
  };
}
```

### 4. Cross-Plugin Communication

```typescript
// In metrics plugin:
ctx.registerService('metricsRegistry', registry);

// In another plugin:
const registry = ctx.getService<Registry>('metricsRegistry');
if (registry) {
  // use registry to create custom metrics
}
```

## CRITICAL Rules
- ALWAYS use `createService()` — never raw `express()` for services
- ALWAYS set plugin `order` to control middleware execution order
- ALWAYS declare `dependencies` if your plugin needs another plugin's services
- ALWAYS implement `dispose()` for plugins that hold resources (connections, timers)
- NEVER call `process.exit()` in plugins — the framework handles graceful shutdown
- Routes marked `skipAuth: true` are automatically added to auth exclude paths

## Checklist
- [ ] Service uses `createService()` builder
- [ ] All plugins have correct `order` values
- [ ] Plugin dependencies declared
- [ ] Resources cleaned up in `dispose()`
- [ ] Health endpoints accessible without auth
- [ ] Metrics endpoint accessible without auth
- [ ] Graceful shutdown working (SIGINT/SIGTERM)
