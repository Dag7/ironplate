import { Command } from 'commander';
import { upCommand, downCommand, statusCommand } from './commands/up';
import {
  profileListCommand,
  profileCreateCommand,
  profileEditCommand,
  profileDeleteCommand,
} from './commands/profile';
import { serviceListCommand, serviceGroupsCommand } from './commands/service';
import type { ProjectConfig } from './config';

export interface TiltManagerConfig {
  projectName: string;
  profilesDir: string;
}

/**
 * Create a configured tilt-manager CLI program.
 * Can be used programmatically or as a standalone CLI.
 */
export function createTiltManagerCLI(config: { name: string; profilesDir: string }): Command {
  const tiltConfig: TiltManagerConfig = {
    projectName: config.name,
    profilesDir: config.profilesDir,
  };

  const program = new Command();

  program
    .name('tilt-manager')
    .description(`Manage Tilt profiles for ${tiltConfig.projectName} local development`)
    .version('0.0.1');

  // Tilt Lifecycle Commands
  program
    .command('up [profile]')
    .description('Start Tilt with a specific profile (default: "default")')
    .option('-f, --force', 'Force restart even if Tilt is already running')
    .option('--no-browser', 'Do not open the Tilt browser UI')
    .action((profileName, options) => upCommand(tiltConfig, profileName, options));

  program
    .command('down')
    .description('Stop Tilt')
    .action(() => downCommand());

  program
    .command('status')
    .description('Show running Tilt services and their statuses')
    .action(() => statusCommand());

  // Profile Commands
  const profile = program.command('profile').description('Manage Tilt profiles');

  profile
    .command('list')
    .alias('ls')
    .description('List available profiles')
    .action(() => profileListCommand(tiltConfig));

  profile
    .command('create <name>')
    .description('Create a new profile interactively')
    .option('-d, --description <desc>', 'Profile description')
    .action((name, options) => profileCreateCommand(tiltConfig, name, options));

  profile
    .command('edit <name>')
    .description('Edit an existing profile')
    .action((name) => profileEditCommand(tiltConfig, name));

  profile
    .command('delete <name>')
    .alias('rm')
    .description('Delete a profile')
    .option('-y, --yes', 'Skip confirmation prompt')
    .action((name, options) => profileDeleteCommand(tiltConfig, name, options));

  // Service Commands
  const service = program.command('service').description('Inspect available Tilt services');

  service
    .command('list')
    .alias('ls')
    .description('List all available services')
    .action(() => serviceListCommand());

  service
    .command('groups')
    .description('List service groups')
    .action(() => serviceGroupsCommand());

  return program;
}
