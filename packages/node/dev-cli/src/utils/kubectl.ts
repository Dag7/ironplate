import { execSync, spawn, ChildProcess, SpawnOptions } from 'child_process';

interface ExecResult {
  stdout: string;
  stderr: string;
}

interface ExecOptions {
  stdio?: SpawnOptions['stdio'];
}

/**
 * Wrapper around kubectl for common Kubernetes operations.
 */
export const kubectl = {
  async exec(args: string[], options?: ExecOptions): Promise<ExecResult> {
    if (options?.stdio === 'inherit') {
      return new Promise((resolve, reject) => {
        const child = spawn('kubectl', args, { stdio: 'inherit' });
        child.on('close', (code) => {
          if (code === 0) {
            resolve({ stdout: '', stderr: '' });
          } else {
            reject(new Error(`kubectl ${args[0]} exited with code ${code}`));
          }
        });
        child.on('error', reject);
      });
    }

    return new Promise((resolve, reject) => {
      try {
        const stdout = execSync(`kubectl ${args.join(' ')}`, {
          encoding: 'utf-8',
          stdio: ['pipe', 'pipe', 'pipe'],
        });
        resolve({ stdout, stderr: '' });
      } catch (err: any) {
        if (err.stdout || err.stderr) {
          reject(new Error(err.stderr?.trim() || err.stdout?.trim() || err.message));
        } else {
          reject(err);
        }
      }
    });
  },

  async getNamespace(): Promise<string> {
    try {
      const result = await kubectl.exec([
        'config', 'view', '--minify',
        '-o', 'jsonpath={.contexts[0].context.namespace}',
      ]);
      return result.stdout.trim() || 'default';
    } catch {
      return 'default';
    }
  },

  async portForward(
    resource: string,
    localPort: number,
    remotePort: number,
    namespace?: string,
  ): Promise<ChildProcess> {
    const args = ['port-forward', resource, `${localPort}:${remotePort}`];
    if (namespace) {
      args.push('-n', namespace);
    }

    const child = spawn('kubectl', args, {
      stdio: ['ignore', 'pipe', 'pipe'],
      detached: false,
    });

    child.on('error', (err) => {
      console.error(`Port-forward error: ${err.message}`);
    });

    child.on('exit', (code) => {
      if (code && code !== 0) {
        console.error(`Port-forward exited with code ${code}`);
      }
    });

    return child;
  },
};
