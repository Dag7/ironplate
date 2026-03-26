import { trace, SpanStatusCode, type Span, type SpanOptions } from '@opentelemetry/api';

/**
 * Creates a traced wrapper around an async function.
 *
 * @example
 * ```ts
 * const result = await withSpan('fetchUser', async (span) => {
 *   span.setAttribute('user.id', userId);
 *   return db.user.findById(userId);
 * });
 * ```
 */
export async function withSpan<T>(
  name: string,
  fn: (span: Span) => Promise<T>,
  options?: SpanOptions,
): Promise<T> {
  const tracer = trace.getTracer('ironplate');
  return tracer.startActiveSpan(name, options ?? {}, async (span) => {
    try {
      const result = await fn(span);
      span.setStatus({ code: SpanStatusCode.OK });
      return result;
    } catch (error) {
      span.setStatus({
        code: SpanStatusCode.ERROR,
        message: error instanceof Error ? error.message : String(error),
      });
      span.recordException(error instanceof Error ? error : new Error(String(error)));
      throw error;
    } finally {
      span.end();
    }
  });
}

/**
 * Get the current active span, if any.
 */
export function getActiveSpan(): Span | undefined {
  return trace.getActiveSpan();
}

/**
 * Add an attribute to the current active span.
 */
export function setSpanAttribute(key: string, value: string | number | boolean): void {
  const span = trace.getActiveSpan();
  if (span) {
    span.setAttribute(key, value);
  }
}
