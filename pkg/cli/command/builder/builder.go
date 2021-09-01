package builder

import (
	"fmt"

	"github.com/rancher/kim/pkg/cli/command/builder/install"
	"github.com/rancher/kim/pkg/cli/command/builder/login"
	"github.com/rancher/kim/pkg/cli/command/builder/uninstall"
	wrangler "github.com/rancher/wrangler-cli"
	"github.com/spf13/cobra"
)

const (
	Short = "Manage Builder(s)"
)

func Use(sub string) string {
	return fmt.Sprintf("%s [OPTIONS] COMMAND", sub)
}

func Command() *cobra.Command {
	cmd := wrangler.Command(&CommandSpec{}, cobra.Command{
		Use:                   Use("builder"),
		Short:                 Short,
		DisableFlagsInUseLine: true,
	})
	cmd.AddCommand(
		install.Command(),
		uninstall.Command(),
		login.Command(),
	)
	return cmd
}

type CommandSpec struct {
}

func (s *CommandSpec) Run(cmd *cobra.Command, _ []string) error {
	return cmd.Help()
}
