import * as path from 'path';
import chalk from 'chalk';
import { parseTiltfile } from '../utils/tilt';

export async function serviceListCommand(): Promise<void> {
  const tiltfilePath = path.resolve(process.cwd(), 'Tiltfile');
  const discovered = parseTiltfile(tiltfilePath);

  const allServices = [
    ...discovered.services.map((s) => ({ ...s, type: 'service' as const })),
    ...discovered.infra.map((name) => ({ name, group: 'infra', type: 'infra' as const })),
  ];

  if (allServices.length === 0) {
    console.log(chalk.yellow('No services discovered from Tiltfile.'));
    console.log(chalk.gray('Make sure you are in the project root with a valid Tiltfile.'));
    return;
  }

  const nameWidth = Math.max(25, ...allServices.map((s) => s.name.length + 2));
  const groupWidth = Math.max(15, ...allServices.map((s) => s.group.length + 2));

  console.log(
    chalk.bold(
      `${'NAME'.padEnd(nameWidth)} ${'GROUP'.padEnd(groupWidth)} TYPE`,
    ),
  );
  console.log('-'.repeat(nameWidth + groupWidth + 12));

  const sorted = allServices.sort((a, b) => {
    if (a.type !== b.type) return a.type === 'infra' ? -1 : 1;
    return a.name.localeCompare(b.name);
  });

  for (const svc of sorted) {
    const typeColor = svc.type === 'infra' ? chalk.cyan : chalk.green;
    console.log(
      `${svc.name.padEnd(nameWidth)} ${chalk.gray(svc.group.padEnd(groupWidth))} ${typeColor(svc.type)}`,
    );
  }

  console.log(
    chalk.gray(
      `\n${discovered.services.length} services, ${discovered.infra.length} infra resources`,
    ),
  );
}

export async function serviceGroupsCommand(): Promise<void> {
  const tiltfilePath = path.resolve(process.cwd(), 'Tiltfile');
  const discovered = parseTiltfile(tiltfilePath);

  if (discovered.services.length === 0) {
    console.log(chalk.yellow('No service groups discovered from Tiltfile.'));
    return;
  }

  const groups = new Map<string, string[]>();
  for (const svc of discovered.services) {
    const existing = groups.get(svc.group) || [];
    existing.push(svc.name);
    groups.set(svc.group, existing);
  }

  if (discovered.infra.length > 0) {
    groups.set('infra', discovered.infra);
  }

  for (const [groupName, members] of Array.from(groups.entries()).sort()) {
    console.log(chalk.bold.blue(`${groupName}`) + chalk.gray(` (${members.length} resources)`));
    for (const member of members.sort()) {
      console.log(`  ${chalk.gray('*')} ${member}`);
    }
    console.log();
  }
}
