import type { Command } from 'commander';

/**
 * Plugin interface for extending the dev CLI with additional commands.
 * Commands register themselves when their required infrastructure component is present.
 */
export interface ICommandPlugin {
  /** Unique plugin name */
  name: string;
  /** Infrastructure component this plugin requires (e.g., 'argocd', 'external-secrets') */
  requiredComponent?: string;
  /** Cloud provider this plugin requires (e.g., 'gcp', 'aws') */
  requiredProvider?: string;
  /** Register commands on the given Commander program */
  register(program: Command, config: ProjectContext): void;
}

export interface ProjectContext {
  /** Project name from ironplate.yaml */
  name: string;
  /** Default Kubernetes namespace */
  namespace: string;
  /** Database name (defaults to project name) */
  dbName: string;
  /** Installed infrastructure components */
  components: string[];
  /** Cloud provider (gcp, aws, azure, none) */
  cloudProvider: string;
}
