/**
 * Creates a factory function that generates test objects with default
 * values that can be overridden per test.
 *
 * @example
 * ```ts
 * const createUser = createFactory({
 *   id: '1',
 *   name: 'Test User',
 *   email: 'test@example.com',
 * });
 *
 * const user = createUser({ name: 'Alice' });
 * // { id: '1', name: 'Alice', email: 'test@example.com' }
 * ```
 */
export function createFactory<T extends Record<string, unknown>>(
  defaults: T,
): (overrides?: Partial<T>) => T {
  let counter = 0;
  return (overrides?: Partial<T>): T => {
    counter += 1;
    return { ...defaults, ...overrides, _seq: counter } as T;
  };
}

/**
 * Creates a sequence generator for unique test values.
 *
 * @example
 * ```ts
 * const nextId = createSequence('user');
 * nextId(); // 'user-1'
 * nextId(); // 'user-2'
 * ```
 */
export function createSequence(prefix: string): () => string {
  let counter = 0;
  return () => {
    counter += 1;
    return `${prefix}-${counter}`;
  };
}
