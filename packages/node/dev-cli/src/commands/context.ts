import { Command } from 'commander';
import chalk from 'chalk';
import { kubectl } from '../utils/kubectl';

export function registerContextCommand(program: Command): void {
  const ctx = program
    .command('context')
    .description('Kubernetes context management');

  ctx
    .command('list')
    .description('List available Kubernetes contexts')
    .action(async () => {
      try {
        const result = await kubectl.exec(['config', 'get-contexts', '-o', 'name']);
        const contexts = result.stdout.trim().split('\n');
        const current = await kubectl.exec(['config', 'current-context']);
        const currentContext = current.stdout.trim();

        console.log(chalk.bold('Available contexts:\n'));
        for (const context of contexts) {
          if (context === currentContext) {
            console.log(chalk.green(`  * ${context} (current)`));
          } else {
            console.log(`    ${context}`);
          }
        }
      } catch (err: any) {
        console.error(chalk.red('Failed to list contexts:'), err.message);
        process.exit(1);
      }
    });

  ctx
    .command('switch <name>')
    .description('Switch to a different Kubernetes context')
    .action(async (name: string) => {
      try {
        await kubectl.exec(['config', 'use-context', name]);
        console.log(chalk.green(`Switched to context: ${name}`));
      } catch (err: any) {
        console.error(chalk.red(`Failed to switch context to "${name}":`), err.message);
        process.exit(1);
      }
    });

  ctx
    .command('current')
    .description('Show the current Kubernetes context')
    .action(async () => {
      try {
        const result = await kubectl.exec(['config', 'current-context']);
        const context = result.stdout.trim();
        console.log(chalk.bold('Current context:'), chalk.cyan(context));

        const ns = await kubectl.getNamespace();
        console.log(chalk.bold('Namespace:'), chalk.cyan(ns));
      } catch (err: any) {
        console.error(chalk.red('Failed to get current context:'), err.message);
        process.exit(1);
      }
    });
}
