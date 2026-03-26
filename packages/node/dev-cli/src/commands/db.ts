import { Command } from 'commander';
import chalk from 'chalk';
import { execSync, ChildProcess } from 'child_process';
import { kubectl } from '../utils/kubectl';
import type { ProjectContext } from '../types';

const DB_LOCAL_PORT = 25432;
const DB_REMOTE_PORT = 5432;
const DB_USER = 'postgres';

let portForwardProcess: ChildProcess | null = null;

async function ensurePortForward(namespace: string): Promise<void> {
  if (portForwardProcess) return;

  console.log(chalk.gray(`Setting up port-forward to database on localhost:${DB_LOCAL_PORT}...`));

  portForwardProcess = await kubectl.portForward(
    'svc/postgres',
    DB_LOCAL_PORT,
    DB_REMOTE_PORT,
    namespace,
  );

  await new Promise((resolve) => setTimeout(resolve, 2000));
  console.log(chalk.green('Port-forward established.'));
}

function stopPortForward(): void {
  if (portForwardProcess) {
    portForwardProcess.kill();
    portForwardProcess = null;
  }
}

export function registerDbCommand(program: Command, config: ProjectContext): void {
  const dbName = config.dbName;

  const db = program
    .command('db')
    .description('Database operations');

  db
    .command('connect')
    .description('Open a psql shell to the dev database')
    .option('-n, --namespace <namespace>', 'Kubernetes namespace', config.namespace)
    .action(async (opts) => {
      try {
        await ensurePortForward(opts.namespace);
        console.log(chalk.bold(`Connecting to database "${dbName}"...\n`));

        const env = {
          ...process.env,
          PGHOST: 'localhost',
          PGPORT: String(DB_LOCAL_PORT),
          PGDATABASE: dbName,
          PGUSER: DB_USER,
        };

        execSync('psql', { env, stdio: 'inherit' });
      } catch (err: any) {
        console.error(chalk.red('Failed to connect to database:'), err.message);
      } finally {
        stopPortForward();
      }
    });

  db
    .command('dump')
    .argument('[file]', 'Output file path', `${dbName}-dump.sql`)
    .description('Dump the database to a file')
    .option('-n, --namespace <namespace>', 'Kubernetes namespace', config.namespace)
    .action(async (file: string, opts) => {
      try {
        await ensurePortForward(opts.namespace);
        console.log(chalk.bold(`Dumping database "${dbName}" to ${file}...`));

        const env = {
          ...process.env,
          PGHOST: 'localhost',
          PGPORT: String(DB_LOCAL_PORT),
          PGDATABASE: dbName,
          PGUSER: DB_USER,
        };

        execSync(`pg_dump --clean --if-exists -f "${file}"`, { env, stdio: 'inherit' });
        console.log(chalk.green(`Database dumped to ${file}`));
      } catch (err: any) {
        console.error(chalk.red('Failed to dump database:'), err.message);
      } finally {
        stopPortForward();
      }
    });

  db
    .command('restore <file>')
    .description('Restore the database from a dump file')
    .option('-n, --namespace <namespace>', 'Kubernetes namespace', config.namespace)
    .action(async (file: string, opts) => {
      try {
        await ensurePortForward(opts.namespace);
        console.log(chalk.bold(`Restoring database "${dbName}" from ${file}...`));

        const env = {
          ...process.env,
          PGHOST: 'localhost',
          PGPORT: String(DB_LOCAL_PORT),
          PGDATABASE: dbName,
          PGUSER: DB_USER,
        };

        execSync(`psql -f "${file}"`, { env, stdio: 'inherit' });
        console.log(chalk.green(`Database restored from ${file}`));
      } catch (err: any) {
        console.error(chalk.red('Failed to restore database:'), err.message);
      } finally {
        stopPortForward();
      }
    });

  db
    .command('reset')
    .description('Drop and recreate the database')
    .option('-n, --namespace <namespace>', 'Kubernetes namespace', config.namespace)
    .option('--yes', 'Skip confirmation prompt', false)
    .action(async (opts) => {
      try {
        if (!opts.yes) {
          const inquirer = await import('inquirer');
          const { confirm } = await inquirer.default.prompt([
            {
              type: 'confirm',
              name: 'confirm',
              message: chalk.yellow(`This will destroy all data in "${dbName}". Continue?`),
              default: false,
            },
          ]);
          if (!confirm) {
            console.log('Aborted.');
            return;
          }
        }

        await ensurePortForward(opts.namespace);
        console.log(chalk.bold(`Resetting database "${dbName}"...`));

        const env = {
          ...process.env,
          PGHOST: 'localhost',
          PGPORT: String(DB_LOCAL_PORT),
          PGUSER: DB_USER,
        };

        execSync(`psql -d postgres -c "DROP DATABASE IF EXISTS \\"${dbName}\\";"`, {
          env,
          stdio: 'inherit',
        });
        execSync(`psql -d postgres -c "CREATE DATABASE \\"${dbName}\\";"`, {
          env,
          stdio: 'inherit',
        });
        console.log(chalk.green(`Database "${dbName}" has been reset.`));
      } catch (err: any) {
        console.error(chalk.red('Failed to reset database:'), err.message);
      } finally {
        stopPortForward();
      }
    });
}
