import { execSync } from 'child_process';
import * as fs from 'fs';

/**
 * Check if a Tilt process is currently running.
 */
export async function isTiltRunning(): Promise<boolean> {
  try {
    execSync('tilt get session 2>/dev/null', { stdio: 'pipe' });
    return true;
  } catch {
    return false;
  }
}

export interface TiltResource {
  name: string;
  status: string;
  type: string;
  updateStatus: string;
}

/**
 * Get the status of all Tilt resources by querying the Tilt API.
 */
export async function getTiltStatus(): Promise<TiltResource[]> {
  try {
    const output = execSync('tilt get uiresources -o json', {
      stdio: ['pipe', 'pipe', 'pipe'],
      encoding: 'utf-8',
    });

    const data = JSON.parse(output);
    if (!data.items || !Array.isArray(data.items)) {
      return [];
    }

    return data.items.map((item: any) => {
      const name = item.metadata?.name || 'unknown';
      const runtimeStatus = item.status?.runtimeStatus || 'unknown';
      const updateStatus = item.status?.updateStatus || 'unknown';
      const labels = item.metadata?.labels || {};
      const type = labels['tilt.dev/resource-type'] || 'unknown';

      return { name, status: runtimeStatus, type, updateStatus };
    });
  } catch {
    return [];
  }
}

export interface DiscoveredService {
  name: string;
  group: string;
}

export interface DiscoveredResources {
  services: DiscoveredService[];
  infra: string[];
}

/**
 * Parse a Tiltfile to discover service and infra resource names.
 * Works with any project — no project-specific assumptions.
 */
export function parseTiltfile(tiltfilePath: string): DiscoveredResources {
  const services: DiscoveredService[] = [];
  const infra: string[] = [];

  if (!fs.existsSync(tiltfilePath)) {
    return { services, infra };
  }

  const content = fs.readFileSync(tiltfilePath, 'utf-8');

  // Extract k8s_resource declarations
  const k8sResourcePattern = /k8s_resource\(\s*['"]([^'"]+)['"]/g;
  let match: RegExpExecArray | null;

  while ((match = k8sResourcePattern.exec(content)) !== null) {
    const name = match[1];
    services.push({ name, group: inferGroup(name, content) });
  }

  // Extract docker_build image names (generic — matches any project/name pattern)
  const dockerBuildPattern = /docker_build\(\s*['"][^/'"]*\/([^'"]+)['"]/g;
  while ((match = dockerBuildPattern.exec(content)) !== null) {
    const name = match[1];
    if (!services.some((s) => s.name === name)) {
      services.push({ name, group: inferGroup(name, content) });
    }
  }

  // Extract infrastructure from load/include calls pointing to infra directories
  const infraLoadPattern = /(?:load|include)\(\s*['"]\.\/k8s\/(?:helm|deployment)\/infra\/([^/'"]+)/g;
  while ((match = infraLoadPattern.exec(content)) !== null) {
    const name = match[1];
    if (!infra.includes(name)) {
      infra.push(name);
    }
  }

  // Extract local_resource names
  const localResourcePattern = /local_resource\(\s*\n?\s*['"]([^'"]+)['"]/g;
  while ((match = localResourcePattern.exec(content)) !== null) {
    const name = match[1];
    if (!infra.includes(name) && !services.some((s) => s.name === name)) {
      infra.push(name);
    }
  }

  // Parse loaded sub-Tiltfiles
  const loadPattern = /load\(\s*['"]([^'"]+Tiltfile)['"]/g;
  while ((match = loadPattern.exec(content)) !== null) {
    const loadedPath = match[1];
    const dir = tiltfilePath.replace(/\/[^/]+$/, '');
    const resolvedPath = loadedPath.startsWith('./')
      ? `${dir}/${loadedPath.slice(2)}`
      : `${dir}/${loadedPath}`;

    if (fs.existsSync(resolvedPath)) {
      const subContent = fs.readFileSync(resolvedPath, 'utf-8');
      const subK8sPattern = /k8s_resource\(\s*['"]([^'"]+)['"]/g;
      let subMatch: RegExpExecArray | null;
      while ((subMatch = subK8sPattern.exec(subContent)) !== null) {
        const name = subMatch[1];
        if (!services.some((s) => s.name === name) && !infra.includes(name)) {
          if (resolvedPath.includes('/infra/')) {
            infra.push(name);
          } else {
            services.push({ name, group: inferGroupFromPath(resolvedPath) });
          }
        }
      }
    }
  }

  return { services, infra };
}

function inferGroup(serviceName: string, tiltfileContent: string): string {
  const helmPathPattern = new RegExp(
    `k8s/helm/(?:apps|services)/([^/'"]+)/.*${escapeRegex(serviceName)}`,
  );
  const helmMatch = helmPathPattern.exec(tiltfileContent);
  if (helmMatch) {
    return helmMatch[1];
  }

  const labelsPattern = new RegExp(
    `['"]${escapeRegex(serviceName)}['"][^)]*labels\\s*=\\s*\\[['"]([^'"]+)['"]`,
  );
  const labelsMatch = labelsPattern.exec(tiltfileContent);
  if (labelsMatch) {
    return labelsMatch[1];
  }

  return 'default';
}

function inferGroupFromPath(tiltfilePath: string): string {
  const match = /k8s\/helm\/(?:apps|services)\/([^/]+)/.exec(tiltfilePath);
  if (match) {
    return match[1];
  }
  return 'default';
}

function escapeRegex(str: string): string {
  return str.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}
