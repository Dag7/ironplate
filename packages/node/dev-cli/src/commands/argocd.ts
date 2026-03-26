import { Command } from 'commander';
import chalk from 'chalk';
import { execSync } from 'child_process';
import type { ProjectContext } from '../types';

export function registerArgoCdCommand(program: Command, config: ProjectContext): void {
  const argocd = program
    .command('argocd')
    .description('ArgoCD application management');

  argocd
    .command('list')
    .description('List ArgoCD applications')
    .action(async () => {
      try {
        const result = execSync('argocd app list --output wide', {
          encoding: 'utf-8',
          stdio: ['pipe', 'pipe', 'pipe'],
        });
        console.log(result);
      } catch (err: any) {
        console.error(chalk.red('Failed to list ArgoCD apps. Is argocd CLI installed and logged in?'));
        console.log(chalk.gray('Login with: argocd login <server>'));
        process.exit(1);
      }
    });

  argocd
    .command('sync <app>')
    .description('Sync an ArgoCD application')
    .option('--prune', 'Allow resource pruning', false)
    .action(async (app: string, opts) => {
      try {
        const args = ['argocd', 'app', 'sync', app];
        if (opts.prune) args.push('--prune');

        console.log(chalk.blue(`Syncing ArgoCD application "${app}"...`));
        execSync(args.join(' '), { stdio: 'inherit' });
        console.log(chalk.green(`Application "${app}" synced successfully.`));
      } catch (err: any) {
        console.error(chalk.red(`Failed to sync application "${app}":`), err.message);
        process.exit(1);
      }
    });

  argocd
    .command('status <app>')
    .description('Show detailed status of an ArgoCD application')
    .action(async (app: string) => {
      try {
        const result = execSync(`argocd app get ${app}`, {
          encoding: 'utf-8',
          stdio: ['pipe', 'pipe', 'pipe'],
        });
        console.log(result);
      } catch (err: any) {
        console.error(chalk.red(`Failed to get status for "${app}":`), err.message);
        process.exit(1);
      }
    });

  argocd
    .command('diff <app>')
    .description('Show diff between live and desired state')
    .action(async (app: string) => {
      try {
        execSync(`argocd app diff ${app}`, { stdio: 'inherit' });
      } catch {
        // argocd diff exits with code 1 when there are differences — that's normal
      }
    });

  argocd
    .command('dashboard')
    .description('Open ArgoCD dashboard (port-forward)')
    .option('-p, --port <port>', 'Local port', '8080')
    .action(async (opts) => {
      try {
        console.log(chalk.blue(`Opening ArgoCD dashboard on http://localhost:${opts.port}...`));
        console.log(chalk.gray('Press Ctrl+C to stop.'));
        execSync(
          `kubectl port-forward svc/argocd-server -n argocd ${opts.port}:443`,
          { stdio: 'inherit' },
        );
      } catch {
        // Normal exit on Ctrl+C
      }
    });
}
