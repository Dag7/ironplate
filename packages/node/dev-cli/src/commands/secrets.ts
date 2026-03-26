import { Command } from 'commander';
import chalk from 'chalk';
import { kubectl } from '../utils/kubectl';
import type { ProjectContext } from '../types';

export function registerSecretsCommand(program: Command, config: ProjectContext): void {
  const secrets = program
    .command('secrets')
    .description('Kubernetes secrets management (via External Secrets Operator)');

  secrets
    .command('list')
    .description('List ExternalSecret resources')
    .option('-n, --namespace <namespace>', 'Kubernetes namespace', config.namespace)
    .action(async (opts) => {
      try {
        const result = await kubectl.exec([
          'get', 'externalsecrets', '-n', opts.namespace,
          '-o', 'custom-columns=NAME:.metadata.name,STORE:.spec.secretStoreRef.name,STATUS:.status.conditions[0].reason,SYNCED:.status.conditions[0].status',
        ]);
        console.log(result.stdout);
      } catch (err: any) {
        console.error(chalk.red('Failed to list secrets:'), err.message);
        console.log(chalk.gray('Make sure External Secrets Operator is installed.'));
        process.exit(1);
      }
    });

  secrets
    .command('sync <name>')
    .description('Force sync an ExternalSecret')
    .option('-n, --namespace <namespace>', 'Kubernetes namespace', config.namespace)
    .action(async (name: string, opts) => {
      try {
        // Annotate to trigger reconciliation
        await kubectl.exec([
          'annotate', 'externalsecrets', name,
          'force-sync=' + Date.now(),
          '--overwrite',
          '-n', opts.namespace,
        ]);
        console.log(chalk.green(`Triggered sync for ExternalSecret "${name}".`));
      } catch (err: any) {
        console.error(chalk.red(`Failed to sync secret "${name}":`), err.message);
        process.exit(1);
      }
    });

  secrets
    .command('status')
    .description('Show status of all secret stores')
    .option('-n, --namespace <namespace>', 'Kubernetes namespace', config.namespace)
    .action(async (opts) => {
      try {
        console.log(chalk.bold('SecretStores:\n'));
        const storeResult = await kubectl.exec([
          'get', 'secretstores', '-n', opts.namespace, '-o', 'wide',
        ]);
        console.log(storeResult.stdout);

        console.log(chalk.bold('\nClusterSecretStores:\n'));
        const clusterResult = await kubectl.exec([
          'get', 'clustersecretstores', '-o', 'wide',
        ]);
        console.log(clusterResult.stdout);
      } catch (err: any) {
        console.error(chalk.red('Failed to get secret store status:'), err.message);
        process.exit(1);
      }
    });
}
