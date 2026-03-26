import { Command } from 'commander';
import type { ICommandPlugin, ProjectContext } from './types';
import { registerContextCommand } from './commands/context';
import { registerPodsCommand } from './commands/pods';
import { registerDbCommand } from './commands/db';
import { registerImagesCommand } from './commands/images';
import { registerSecretsCommand } from './commands/secrets';
import { registerArgoCdCommand } from './commands/argocd';

export type DevCLIConfig = ProjectContext;

/** All built-in command plugins */
const builtinPlugins: ICommandPlugin[] = [
  // Core commands — always registered
  {
    name: 'context',
    register: (program, config) => registerContextCommand(program),
  },
  {
    name: 'pods',
    register: (program, config) => registerPodsCommand(program),
  },
  {
    name: 'db',
    register: (program, config) => registerDbCommand(program, config),
  },
  {
    name: 'images',
    register: (program, config) => registerImagesCommand(program),
  },
  // Conditional commands — only when component is present
  {
    name: 'secrets',
    requiredComponent: 'external-secrets',
    register: (program, config) => registerSecretsCommand(program, config),
  },
  {
    name: 'argocd',
    requiredComponent: 'argocd',
    register: (program, config) => registerArgoCdCommand(program, config),
  },
];

/**
 * Create a configured developer CLI program.
 * Automatically discovers which commands to enable based on project config.
 */
export function createDevCLI(config: ProjectContext, extraPlugins?: ICommandPlugin[]): Command {
  const program = new Command();

  program
    .name(config.name)
    .description(`Developer CLI for ${config.name}`)
    .version('0.0.1');

  const allPlugins = [...builtinPlugins, ...(extraPlugins || [])];

  for (const plugin of allPlugins) {
    // Skip plugins whose required component is not installed
    if (plugin.requiredComponent && !config.components.includes(plugin.requiredComponent)) {
      continue;
    }
    // Skip plugins whose required cloud provider doesn't match
    if (plugin.requiredProvider && config.cloudProvider !== plugin.requiredProvider) {
      continue;
    }
    plugin.register(program, config);
  }

  return program;
}
