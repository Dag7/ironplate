import type { Request, Response, Router } from 'express';
import { createLogger } from '@ironplate/logger';
import type { EventBus } from './eventbus';
import type { EventEnvelope } from './types';

/**
 * Creates Express middleware that registers Dapr subscription endpoints.
 *
 * This sets up:
 * - GET /dapr/subscribe — returns the subscription list for the Dapr runtime.
 * - POST /events/:topic — receives published events and dispatches to registered handlers.
 *
 * @param router - An Express Router instance.
 * @param eventBus - The EventBus instance with registered handlers.
 * @returns The router with subscription routes attached.
 */
export function createSubscriptionMiddleware(router: Router, eventBus: EventBus): Router {
  const logger = createLogger('eventbus:middleware');

  // Dapr calls this endpoint to discover subscriptions.
  router.get('/dapr/subscribe', (_req: Request, res: Response) => {
    const subscriptions = eventBus.getTopics().map((topic) => ({
      pubsubname: eventBus.getPubsubName(),
      topic,
      route: `/events/${topic}`,
    }));

    logger.debug({ subscriptions }, 'Returning Dapr subscription list');
    res.json(subscriptions);
  });

  // Dapr delivers events to this endpoint.
  router.post('/events/:topic', async (req: Request, res: Response) => {
    const topic = req.params.topic as string;
    const envelope = req.body as EventEnvelope<unknown>;

    const handlers = eventBus.getHandlers(topic);
    if (handlers.length === 0) {
      logger.warn({ topic }, 'No handlers registered for topic');
      // Return SUCCESS to Dapr so it does not retry unhandled topics.
      res.json({ status: 'SUCCESS' });
      return;
    }

    try {
      logger.debug({ topic, eventId: envelope.id }, 'Dispatching event to handlers');
      await Promise.all(handlers.map((handler) => handler(envelope)));
      logger.info({ topic, eventId: envelope.id }, 'Event processed successfully');
      res.json({ status: 'SUCCESS' });
    } catch (error) {
      const message = error instanceof Error ? error.message : String(error);
      logger.error({ topic, eventId: envelope.id, error: message }, 'Event processing failed');
      // Signal Dapr to retry delivery.
      res.json({ status: 'RETRY' });
    }
  });

  return router;
}
