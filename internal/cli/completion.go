package cli

import (
	"os"

	"github.com/spf13/cobra"
)

func newCompletionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate shell completion scripts",
		Long: `Generate shell completion scripts for iron.

To load completions:

  bash:
    $ source <(iron completion bash)

    # To load completions for each session, execute once:
    # Linux:
    $ iron completion bash > /etc/bash_completion.d/iron
    # macOS:
    $ iron completion bash > $(brew --prefix)/etc/bash_completion.d/iron

  zsh:
    # If shell completion is not already enabled in your environment,
    # you will need to enable it. You can execute the following once:
    $ echo "autoload -U compinit; compinit" >> ~/.zshrc

    # To load completions for each session, execute once:
    $ iron completion zsh > "${fpath[1]}/_iron"

    # You will need to start a new shell for this setup to take effect.

  fish:
    $ iron completion fish | source

    # To load completions for each session, execute once:
    $ iron completion fish > ~/.config/fish/completions/iron.fish

  powershell:
    PS> iron completion powershell | Out-String | Invoke-Expression

    # To load completions for every new session, run:
    PS> iron completion powershell > iron.ps1
    # and source this file from your PowerShell profile.
`,
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return cmd.Root().GenBashCompletionV2(os.Stdout, true)
			case "zsh":
				return cmd.Root().GenZshCompletion(os.Stdout)
			case "fish":
				return cmd.Root().GenFishCompletion(os.Stdout, true)
			case "powershell":
				return cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
			}
			return nil
		},
	}

	return cmd
}
