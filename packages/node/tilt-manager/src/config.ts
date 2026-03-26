import * as fs from 'fs';
import * as path from 'path';
import * as YAML from 'yaml';

export interface ProjectConfig {
  name: string;
  profilesDir: string;
}

/**
 * Resolve project configuration by reading ironplate.yaml or package.json
 * from the current working directory.
 */
export function resolveProjectConfig(): ProjectConfig {
  const cwd = process.cwd();

  // Try ironplate.yaml first
  const ironplatePath = path.join(cwd, 'ironplate.yaml');
  if (fs.existsSync(ironplatePath)) {
    const content = fs.readFileSync(ironplatePath, 'utf-8');
    const config = YAML.parse(content);
    return {
      name: config?.metadata?.name || path.basename(cwd),
      profilesDir: path.join(cwd, '.tilt-profiles'),
    };
  }

  // Fall back to package.json
  const pkgPath = path.join(cwd, 'package.json');
  if (fs.existsSync(pkgPath)) {
    const pkg = JSON.parse(fs.readFileSync(pkgPath, 'utf-8'));
    return {
      name: pkg.name || path.basename(cwd),
      profilesDir: path.join(cwd, '.tilt-profiles'),
    };
  }

  // Default
  return {
    name: path.basename(cwd),
    profilesDir: path.join(cwd, '.tilt-profiles'),
  };
}
