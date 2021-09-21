package completion

import (
	"errors"

	"github.com/spf13/cobra"
	"github.com/stackrox/rox/roxctl/common"
)

var (
	errInvalidArgs = errors.New("Invalid argument. Use one of the following: [bash|zsh|fish|powershell]")
)

const (
	longDescriptionForCompletion = `To load completions:

Bash:
  # Preparations on MacOS
	- Make sure that you are using bash version 4.1 or newer
	- You must install and configure bash-completion v2
	- You must reload your shell after you install bash-completion

  # Preparations on Linux
	- Make sure that you have installed bash-completion. You can install the package by using your distribution's
      package manager.

  $ source <(roxctl completion bash)

  # To load completions for each session, run the following command once:
  # Linux:
  $ roxctl completion bash | sudo cp /dev/stdin /etc/bash_completion.d/roxctl
  # macOS:
  $ roxctl completion bash > /usr/local/etc/bash_completion.d/roxctl

Zsh:

  # To enable compinit for shell completion, run the following command once:

  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # To load completions for each session, run the following command once:
  $ roxctl completion zsh > "${fpath[1]}/_roxctl"

  # You must start a new shell to use shell-completion in zsh.

fish:

  $ roxctl completion fish | source

  # To load completions for each session, run the following command once:
  $ roxctl completion fish > ~/.config/fish/completions/roxctl.fish

PowerShell:

  PS> roxctl completion powershell | Out-String | Invoke-Expression

  # To load completions for every new session, run the following command:
  PS> roxctl completion powershell > roxctl.ps1
  # and source this file from your PowerShell profile.`
)

// Command provides the shell completion cobra command
func Command() *cobra.Command {
	return &cobra.Command{
		DisableFlagsInUseLine: true,
		Use:                   "completion [bash|zsh|fish|powershell]",
		Long:                  longDescriptionForCompletion,
		Args:                  common.ExactArgsWithCustomErrMessage(1, "Missing argument. Use one of the following: [bash|zsh|fish|powershell]"),
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return cmd.Root().GenBashCompletion(cmd.OutOrStdout())
			case "zsh":
				return cmd.Root().GenZshCompletion(cmd.OutOrStdout())
			case "fish":
				return cmd.Root().GenFishCompletion(cmd.OutOrStdout(), true)
			case "powershell":
				return cmd.Root().GenPowerShellCompletionWithDesc(cmd.OutOrStdout())
			default:
				return errInvalidArgs
			}
		},
	}
}
