package completion

import (
	"os"

	"github.com/spf13/cobra"
)

func Command() *cobra.Command {
	completionCmd := &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate completion script",
		Long: `To load completions:
Bash:
  $ source <(kim completion bash)
  # To load completions for each session, execute once:
  # Linux:
  $ kim completion bash > /etc/bash_completion.d/kim
  # macOS:
  $ kim completion bash > /usr/local/etc/bash_completion.d/kim
Zsh:
  # If shell completion is not already enabled in your environment,
  # you will need to enable it.  You can execute the following once:
  $ echo "autoload -U compinit; compinit" >> ~/.zshrc
  # To load completions for each session, execute once:
  $ kim completion zsh > "${fpath[1]}/_kim"
  # You will need to start a new shell for this setup to take effect.
fish:
  $ kim completion fish | source
  # To load completions for each session, execute once:
  $ kim completion fish > ~/.config/fish/completions/kim.fish
PowerShell:
  PS> kim completion powershell | Out-String | Invoke-Expression
  # To load completions for every new session, run:
  PS> kim completion powershell > kim.ps1
  # and source this file from your PowerShell profile.
`,
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.ExactValidArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			switch args[0] {
			case "bash":
				_ = cmd.Root().GenBashCompletion(os.Stdout)
			case "zsh":
				_ = cmd.Root().GenZshCompletion(os.Stdout)
			case "fish":
				_ = cmd.Root().GenFishCompletion(os.Stdout, true)
			case "powershell":
				_ = cmd.Root().GenPowerShellCompletion(os.Stdout)
			}
		},
	}

	return completionCmd
}
