#!/usr/bin/env node

export { createTiltManagerCLI, type TiltManagerConfig } from './cli';
export { parseTiltfile, type DiscoveredResources, type DiscoveredService } from './utils/tilt';
export { isTiltRunning, getTiltStatus, type TiltResource } from './utils/tilt';

// When run directly as a CLI, auto-detect project and start
if (require.main === module) {
  const { createTiltManagerCLI } = require('./cli');
  const { resolveProjectConfig } = require('./config');

  const config = resolveProjectConfig();
  const cli = createTiltManagerCLI(config);
  cli.parse(process.argv);
}
