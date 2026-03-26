import { Command } from 'commander';
import chalk from 'chalk';
import { execSync } from 'child_process';

const REGISTRY_HOST = 'localhost:5050';

export function registerImagesCommand(program: Command): void {
  const images = program
    .command('images')
    .description('Container image inspection and management');

  images
    .command('list')
    .description('List images in the local registry')
    .action(async () => {
      try {
        const result = execSync(
          `curl -s http://${REGISTRY_HOST}/v2/_catalog`,
          { encoding: 'utf-8' },
        );
        const catalog = JSON.parse(result);
        const repos: string[] = catalog.repositories || [];

        if (repos.length === 0) {
          console.log(chalk.yellow('No images found in the local registry.'));
          return;
        }

        console.log(chalk.bold('Images in local registry:\n'));
        for (const repo of repos) {
          const tagsResult = execSync(
            `curl -s http://${REGISTRY_HOST}/v2/${repo}/tags/list`,
            { encoding: 'utf-8' },
          );
          const tagsData = JSON.parse(tagsResult);
          const tags: string[] = tagsData.tags || [];

          console.log(chalk.cyan(`  ${repo}`));
          for (const tag of tags) {
            console.log(chalk.gray(`    - ${tag}`));
          }
        }
      } catch (err: any) {
        console.error(chalk.red('Failed to list images. Is the local registry running?'), err.message);
        process.exit(1);
      }
    });

  images
    .command('inspect <name>')
    .description('Show details for an image in the local registry')
    .option('-t, --tag <tag>', 'Image tag to inspect', 'latest')
    .action(async (name: string, opts) => {
      try {
        const manifestResult = execSync(
          `curl -s -H "Accept: application/vnd.docker.distribution.manifest.v2+json" http://${REGISTRY_HOST}/v2/${name}/manifests/${opts.tag}`,
          { encoding: 'utf-8' },
        );
        const manifest = JSON.parse(manifestResult);

        console.log(chalk.bold(`Image: ${name}:${opts.tag}\n`));
        console.log(chalk.gray('Schema Version:'), manifest.schemaVersion);

        if (manifest.config) {
          console.log(chalk.gray('Config Digest: '), manifest.config.digest);
          console.log(chalk.gray('Config Size:   '), formatBytes(manifest.config.size));
        }

        if (manifest.layers) {
          console.log(chalk.gray('\nLayers:'));
          let totalSize = 0;
          for (const layer of manifest.layers) {
            console.log(chalk.gray(`  - ${layer.digest} (${formatBytes(layer.size)})`));
            totalSize += layer.size;
          }
          console.log(chalk.bold(`\nTotal size: ${formatBytes(totalSize)}`));
        }
      } catch (err: any) {
        console.error(chalk.red(`Failed to inspect image "${name}":`), err.message);
        process.exit(1);
      }
    });

  images
    .command('prune')
    .description('Clean up old/untagged images from the local registry')
    .option('--yes', 'Skip confirmation prompt', false)
    .action(async (opts) => {
      try {
        if (!opts.yes) {
          const inquirer = await import('inquirer');
          const { confirm } = await inquirer.default.prompt([
            {
              type: 'confirm',
              name: 'confirm',
              message: chalk.yellow('This will remove untagged images from the local registry. Continue?'),
              default: false,
            },
          ]);
          if (!confirm) {
            console.log('Aborted.');
            return;
          }
        }

        console.log(chalk.bold('Pruning untagged images from the local registry...'));

        const registryContainer = execSync(
          'docker ps --filter "name=registry" --format "{{.Names}}"',
          { encoding: 'utf-8' },
        ).trim();

        if (!registryContainer) {
          console.log(chalk.yellow('No registry container found. Is the local registry running?'));
          return;
        }

        execSync(
          `docker exec ${registryContainer} bin/registry garbage-collect /etc/docker/registry/config.yml --delete-untagged=true`,
          { stdio: 'inherit' },
        );

        console.log(chalk.green('Registry garbage collection complete.'));
      } catch (err: any) {
        console.error(chalk.red('Failed to prune images:'), err.message);
        process.exit(1);
      }
    });
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B';
  const units = ['B', 'KB', 'MB', 'GB'];
  const i = Math.floor(Math.log(bytes) / Math.log(1024));
  const value = bytes / Math.pow(1024, i);
  return `${value.toFixed(1)} ${units[i]}`;
}
