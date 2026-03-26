import * as fs from 'fs';
import * as path from 'path';
import chalk from 'chalk';
import * as YAML from 'yaml';
import inquirer from 'inquirer';
import { parseTiltfile } from '../utils/tilt';
import type { TiltManagerConfig } from '../cli';

interface Profile {
  name: string;
  description?: string;
  services: string[];
  infra: string[];
}

function ensureProfilesDir(profilesDir: string): void {
  if (!fs.existsSync(profilesDir)) {
    fs.mkdirSync(profilesDir, { recursive: true });
  }
}

function getProfilePath(profilesDir: string, name: string): string {
  return path.join(profilesDir, `${name}.yaml`);
}

function loadAllProfiles(profilesDir: string): Profile[] {
  ensureProfilesDir(profilesDir);
  const files = fs.readdirSync(profilesDir).filter((f) => f.endsWith('.yaml'));
  return files.map((f) => {
    const content = fs.readFileSync(path.join(profilesDir, f), 'utf-8');
    return YAML.parse(content) as Profile;
  });
}

function saveProfile(profilesDir: string, profile: Profile): void {
  ensureProfilesDir(profilesDir);
  const profilePath = getProfilePath(profilesDir, profile.name);
  const content = YAML.stringify(profile, { indent: 2 });
  fs.writeFileSync(profilePath, content, 'utf-8');
}

export async function profileListCommand(config: TiltManagerConfig): Promise<void> {
  const profiles = loadAllProfiles(config.profilesDir);

  if (profiles.length === 0) {
    console.log(chalk.yellow('No profiles found.'));
    console.log(chalk.gray('Create one with: tilt-manager profile create <name>'));
    return;
  }

  const nameWidth = Math.max(20, ...profiles.map((p) => p.name.length + 2));

  console.log(
    chalk.bold(
      `${'NAME'.padEnd(nameWidth)} ${'SERVICES'.padEnd(10)} ${'INFRA'.padEnd(10)} DESCRIPTION`,
    ),
  );
  console.log('-'.repeat(nameWidth + 64));

  for (const profile of profiles) {
    const desc = profile.description || chalk.gray('(none)');
    console.log(
      `${profile.name.padEnd(nameWidth)} ${String(profile.services.length).padEnd(10)} ${String(profile.infra.length).padEnd(10)} ${desc}`,
    );
  }
}

export async function profileCreateCommand(
  config: TiltManagerConfig,
  name: string,
  options: { description?: string },
): Promise<void> {
  const profilePath = getProfilePath(config.profilesDir, name);
  if (fs.existsSync(profilePath)) {
    console.error(chalk.red(`Profile "${name}" already exists. Use "profile edit" instead.`));
    process.exit(1);
  }

  console.log(chalk.blue(`Creating profile: ${chalk.bold(name)}\n`));

  const tiltfilePath = path.resolve(process.cwd(), 'Tiltfile');
  const discovered = parseTiltfile(tiltfilePath);

  if (discovered.services.length === 0 && discovered.infra.length === 0) {
    console.log(chalk.yellow('No services discovered from Tiltfile.'));
    console.log(chalk.gray('Make sure you are in the project root with a valid Tiltfile.'));
    process.exit(1);
  }

  const infraAnswers = await inquirer.prompt([
    {
      type: 'checkbox',
      name: 'infra',
      message: 'Select infrastructure resources to include:',
      choices: discovered.infra.map((name) => ({ name, checked: true })),
    },
  ]);

  const serviceAnswers = await inquirer.prompt([
    {
      type: 'checkbox',
      name: 'services',
      message: 'Select application services to include:',
      choices: discovered.services.map((s) => ({
        name: `${s.name} ${chalk.gray(`(${s.group})`)}`,
        value: s.name,
        checked: false,
      })),
    },
  ]);

  let description = options.description;
  if (!description) {
    const descAnswer = await inquirer.prompt([
      {
        type: 'input',
        name: 'description',
        message: 'Profile description (optional):',
      },
    ]);
    description = descAnswer.description || undefined;
  }

  const profile: Profile = {
    name,
    description,
    services: serviceAnswers.services,
    infra: infraAnswers.infra,
  };

  saveProfile(config.profilesDir, profile);

  const totalResources = profile.services.length + profile.infra.length;
  console.log(
    chalk.green(
      `\nProfile "${name}" created with ${totalResources} resources (${profile.services.length} services, ${profile.infra.length} infra).`,
    ),
  );
}

export async function profileEditCommand(config: TiltManagerConfig, name: string): Promise<void> {
  const profilePath = getProfilePath(config.profilesDir, name);
  if (!fs.existsSync(profilePath)) {
    console.error(chalk.red(`Profile "${name}" does not exist.`));
    process.exit(1);
  }

  const content = fs.readFileSync(profilePath, 'utf-8');
  const profile = YAML.parse(content) as Profile;

  console.log(chalk.blue(`Editing profile: ${chalk.bold(name)}\n`));

  const tiltfilePath = path.resolve(process.cwd(), 'Tiltfile');
  const discovered = parseTiltfile(tiltfilePath);

  const infraAnswers = await inquirer.prompt([
    {
      type: 'checkbox',
      name: 'infra',
      message: 'Select infrastructure resources to include:',
      choices: discovered.infra.map((infraName) => ({
        name: infraName,
        checked: profile.infra.includes(infraName),
      })),
    },
  ]);

  const serviceAnswers = await inquirer.prompt([
    {
      type: 'checkbox',
      name: 'services',
      message: 'Select application services to include:',
      choices: discovered.services.map((s) => ({
        name: `${s.name} ${chalk.gray(`(${s.group})`)}`,
        value: s.name,
        checked: profile.services.includes(s.name),
      })),
    },
  ]);

  const descAnswer = await inquirer.prompt([
    {
      type: 'input',
      name: 'description',
      message: 'Profile description:',
      default: profile.description || '',
    },
  ]);

  const updatedProfile: Profile = {
    name,
    description: descAnswer.description || undefined,
    services: serviceAnswers.services,
    infra: infraAnswers.infra,
  };

  saveProfile(config.profilesDir, updatedProfile);

  const totalResources = updatedProfile.services.length + updatedProfile.infra.length;
  console.log(
    chalk.green(
      `\nProfile "${name}" updated with ${totalResources} resources.`,
    ),
  );
}

export async function profileDeleteCommand(
  config: TiltManagerConfig,
  name: string,
  options: { yes?: boolean },
): Promise<void> {
  const profilePath = getProfilePath(config.profilesDir, name);
  if (!fs.existsSync(profilePath)) {
    console.error(chalk.red(`Profile "${name}" does not exist.`));
    process.exit(1);
  }

  if (name === 'default') {
    console.error(chalk.red('Cannot delete the default profile.'));
    process.exit(1);
  }

  if (!options.yes) {
    const confirm = await inquirer.prompt([
      {
        type: 'confirm',
        name: 'proceed',
        message: `Are you sure you want to delete profile "${name}"?`,
        default: false,
      },
    ]);

    if (!confirm.proceed) {
      console.log(chalk.gray('Cancelled.'));
      return;
    }
  }

  fs.unlinkSync(profilePath);
  console.log(chalk.green(`Profile "${name}" deleted.`));
}
