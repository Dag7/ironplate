import { spawn } from 'child_process';
import * as fs from 'fs';
import * as path from 'path';
import chalk from 'chalk';
import * as YAML from 'yaml';
import { isTiltRunning, getTiltStatus } from '../utils/tilt';
import type { TiltManagerConfig } from '../cli';

interface Profile {
  name: string;
  description?: string;
  services: string[];
  infra: string[];
}

function loadProfile(profilesDir: string, profileName: string): Profile | null {
  const profilePath = path.join(profilesDir, `${profileName}.yaml`);
  if (!fs.existsSync(profilePath)) {
    return null;
  }
  const content = fs.readFileSync(profilePath, 'utf-8');
  return YAML.parse(content) as Profile;
}

function buildTiltArgs(profile: Profile, options: { noBrowser?: boolean }): string[] {
  const args: string[] = ['up'];

  const allResources = [...profile.services, ...profile.infra];
  for (const resource of allResources) {
    args.push('--only', resource);
  }

  if (options.noBrowser) {
    args.push('--no-browser');
  }

  return args;
}

export async function upCommand(
  config: TiltManagerConfig,
  profileName: string = 'default',
  options: { force?: boolean; browser?: boolean },
): Promise<void> {
  console.log(chalk.blue(`Starting Tilt with profile: ${chalk.bold(profileName)}`));

  if (!options.force && (await isTiltRunning())) {
    console.log(
      chalk.yellow('Tilt is already running. Use --force to restart, or run "tilt-manager down" first.'),
    );
    process.exit(1);
  }

  const profile = loadProfile(config.profilesDir, profileName);
  if (!profile) {
    console.error(chalk.red(`Profile "${profileName}" not found.`));
    console.log(chalk.gray(`Available profiles are in ${config.profilesDir}`));
    console.log(chalk.gray('Run "tilt-manager profile list" to see available profiles.'));
    process.exit(1);
  }

  const totalResources = profile.services.length + profile.infra.length;
  console.log(
    chalk.gray(
      `Profile "${profile.name}": ${profile.services.length} services, ${profile.infra.length} infra (${totalResources} total resources)`,
    ),
  );

  if (profile.description) {
    console.log(chalk.gray(`Description: ${profile.description}`));
  }

  const noBrowser = options.browser === false;
  const tiltArgs = buildTiltArgs(profile, { noBrowser });

  console.log(chalk.gray(`> tilt ${tiltArgs.join(' ')}`));
  console.log();

  const tilt = spawn('tilt', tiltArgs, {
    stdio: 'inherit',
    cwd: process.cwd(),
  });

  tilt.on('error', (err) => {
    if ((err as NodeJS.ErrnoException).code === 'ENOENT') {
      console.error(chalk.red('Error: tilt is not installed or not in PATH.'));
      console.log(chalk.gray('Install Tilt: https://docs.tilt.dev/install.html'));
    } else {
      console.error(chalk.red(`Error starting Tilt: ${err.message}`));
    }
    process.exit(1);
  });

  tilt.on('exit', (code) => {
    process.exit(code ?? 0);
  });
}

export async function downCommand(): Promise<void> {
  console.log(chalk.blue('Stopping Tilt...'));

  const tilt = spawn('tilt', ['down'], {
    stdio: 'inherit',
    cwd: process.cwd(),
  });

  tilt.on('error', (err) => {
    console.error(chalk.red(`Error stopping Tilt: ${err.message}`));
    process.exit(1);
  });

  tilt.on('exit', (code) => {
    if (code === 0) {
      console.log(chalk.green('Tilt stopped successfully.'));
    }
    process.exit(code ?? 0);
  });
}

export async function statusCommand(): Promise<void> {
  const running = await isTiltRunning();
  if (!running) {
    console.log(chalk.yellow('Tilt is not running.'));
    return;
  }

  console.log(chalk.blue('Fetching Tilt resource statuses...\n'));

  const resources = await getTiltStatus();
  if (!resources || resources.length === 0) {
    console.log(chalk.yellow('No resources found.'));
    return;
  }

  const nameWidth = Math.max(20, ...resources.map((r) => r.name.length + 2));

  console.log(
    chalk.bold(
      `${'NAME'.padEnd(nameWidth)} ${'STATUS'.padEnd(12)} ${'TYPE'.padEnd(12)} UPDATE`,
    ),
  );
  console.log('-'.repeat(nameWidth + 40));

  for (const resource of resources) {
    const statusColor =
      resource.status === 'ok'
        ? chalk.green
        : resource.status === 'pending'
          ? chalk.yellow
          : resource.status === 'error'
            ? chalk.red
            : chalk.gray;

    console.log(
      `${resource.name.padEnd(nameWidth)} ${statusColor(resource.status.padEnd(12))} ${resource.type.padEnd(12)} ${resource.updateStatus}`,
    );
  }
}
