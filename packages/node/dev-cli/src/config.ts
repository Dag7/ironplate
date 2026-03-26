import * as fs from 'fs';
import * as path from 'path';
import * as YAML from 'yaml';
import type { ProjectContext } from './types';

/**
 * Resolve project configuration from ironplate.yaml in the current working directory.
 */
export function resolveProjectConfig(): ProjectContext {
  const cwd = process.cwd();
  const ironplatePath = path.join(cwd, 'ironplate.yaml');

  if (fs.existsSync(ironplatePath)) {
    const content = fs.readFileSync(ironplatePath, 'utf-8');
    const config = YAML.parse(content);

    return {
      name: config?.metadata?.name || path.basename(cwd),
      namespace: config?.metadata?.name || 'default',
      dbName: config?.spec?.infrastructure?.database
        ? config.metadata.name
        : path.basename(cwd),
      components: config?.spec?.infrastructure?.components || [],
      cloudProvider: config?.spec?.cloud?.provider || 'none',
    };
  }

  // Fallback for non-ironplate projects
  return {
    name: path.basename(cwd),
    namespace: 'default',
    dbName: path.basename(cwd),
    components: [],
    cloudProvider: 'none',
  };
}
