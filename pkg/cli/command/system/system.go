package system

import (
	"github.com/rancher/kim/pkg/cli/command/system/info"
	"github.com/rancher/kim/pkg/cli/command/system/install"
	"github.com/rancher/kim/pkg/cli/command/system/uninstall"
	wrangler "github.com/rancher/wrangler-cli"
	"github.com/spf13/cobra"
)

func Command() *cobra.Command {
	cmd := wrangler.Command(&CommandSpec{}, cobra.Command{
		Use:                   "system [OPTIONS] COMMAND",
		Short:                 "Manage KIM",
		DisableFlagsInUseLine: true,
	})
	cmd.AddCommand(
		info.Command(),
		install.Command(),
		uninstall.Command(),
	)
	return cmd
}

type CommandSpec struct {
}

func (s *CommandSpec) Run(cmd *cobra.Command, _ []string) error {
	return cmd.Help()
}
