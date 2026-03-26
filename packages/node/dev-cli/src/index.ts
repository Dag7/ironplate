#!/usr/bin/env node

export { createDevCLI, type DevCLIConfig } from './cli';
export { kubectl } from './utils/kubectl';
export type { ICommandPlugin } from './types';

// When run directly as a CLI, auto-detect project and start
if (require.main === module) {
  const { createDevCLI } = require('./cli');
  const { resolveProjectConfig } = require('./config');

  const config = resolveProjectConfig();
  const cli = createDevCLI(config);
  cli.parse(process.argv);
}
