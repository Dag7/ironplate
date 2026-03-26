import { Command } from 'commander';
import chalk from 'chalk';
import { kubectl } from '../utils/kubectl';

export function registerPodsCommand(program: Command): void {
  const pods = program
    .command('pods')
    .description('Pod debugging helpers');

  pods
    .command('list')
    .argument('[namespace]', 'Kubernetes namespace', 'default')
    .description('List pods with status')
    .action(async (namespace: string) => {
      try {
        const result = await kubectl.exec([
          'get', 'pods', '-n', namespace, '-o', 'wide', '--no-headers',
        ]);

        const lines = result.stdout.trim().split('\n').filter(Boolean);
        if (lines.length === 0) {
          console.log(chalk.yellow(`No pods found in namespace "${namespace}".`));
          return;
        }

        console.log(chalk.bold(`Pods in namespace "${namespace}":\n`));
        console.log(
          chalk.gray(
            'NAME'.padEnd(50) + 'READY'.padEnd(10) + 'STATUS'.padEnd(15) + 'RESTARTS'.padEnd(12) + 'AGE',
          ),
        );
        console.log(chalk.gray('-'.repeat(100)));

        for (const line of lines) {
          const parts = line.trim().split(/\s+/);
          const [name, ready, status, restarts, age] = parts;

          const statusColor =
            status === 'Running' ? chalk.green
            : status === 'Completed' ? chalk.blue
            : status === 'Pending' ? chalk.yellow
            : chalk.red;

          console.log(
            name.padEnd(50) + ready.padEnd(10) + statusColor(status.padEnd(15)) + restarts.padEnd(12) + age,
          );
        }
      } catch (err: any) {
        console.error(chalk.red('Failed to list pods:'), err.message);
        process.exit(1);
      }
    });

  pods
    .command('logs <pod>')
    .description('Stream logs from a pod')
    .option('-n, --namespace <namespace>', 'Kubernetes namespace', 'default')
    .option('-c, --container <container>', 'Container name')
    .option('-f, --follow', 'Follow log output', false)
    .option('--tail <lines>', 'Number of recent lines to show', '100')
    .action(async (pod: string, opts) => {
      try {
        const args = ['logs', pod, '-n', opts.namespace, '--tail', opts.tail];
        if (opts.container) args.push('-c', opts.container);
        if (opts.follow) args.push('-f');

        console.log(chalk.bold(`Logs for pod "${pod}":\n`));
        await kubectl.exec(args, { stdio: 'inherit' });
      } catch (err: any) {
        console.error(chalk.red(`Failed to get logs for pod "${pod}":`), err.message);
        process.exit(1);
      }
    });

  pods
    .command('exec <pod>')
    .argument('[cmd...]', 'Command to execute', ['sh'])
    .description('Execute a command in a pod')
    .option('-n, --namespace <namespace>', 'Kubernetes namespace', 'default')
    .option('-c, --container <container>', 'Container name')
    .action(async (pod: string, cmd: string[], opts) => {
      try {
        const shellCmd = cmd.length > 0 ? cmd : ['sh'];
        const args = ['exec', '-it', pod, '-n', opts.namespace];
        if (opts.container) args.push('-c', opts.container);
        args.push('--', ...shellCmd);

        await kubectl.exec(args, { stdio: 'inherit' });
      } catch (err: any) {
        console.error(chalk.red(`Failed to exec into pod "${pod}":`), err.message);
        process.exit(1);
      }
    });

  pods
    .command('restart <deployment>')
    .description('Rollout restart a deployment')
    .option('-n, --namespace <namespace>', 'Kubernetes namespace', 'default')
    .action(async (deployment: string, opts) => {
      try {
        await kubectl.exec([
          'rollout', 'restart', `deployment/${deployment}`, '-n', opts.namespace,
        ]);
        console.log(chalk.green(`Restarted deployment "${deployment}" in namespace "${opts.namespace}".`));
      } catch (err: any) {
        console.error(chalk.red(`Failed to restart deployment "${deployment}":`), err.message);
        process.exit(1);
      }
    });
}
