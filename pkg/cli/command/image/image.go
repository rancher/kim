package image

import (
	"fmt"

	"github.com/rancher/kim/pkg/cli/command/image/build"
	"github.com/rancher/kim/pkg/cli/command/image/list"
	"github.com/rancher/kim/pkg/cli/command/image/pull"
	"github.com/rancher/kim/pkg/cli/command/image/push"
	"github.com/rancher/kim/pkg/cli/command/image/remove"
	"github.com/rancher/kim/pkg/cli/command/image/tag"
	wrangler "github.com/rancher/wrangler-cli"
	"github.com/spf13/cobra"
)

const (
	Short = "Manage Images"
)

func Use(sub string) string {
	return fmt.Sprintf("%s [OPTIONS] COMMAND", sub)
}

func Command() *cobra.Command {
	cmd := wrangler.Command(&CommandSpec{}, cobra.Command{
		Use:                   Use("image"),
		Short:                 Short,
		DisableFlagsInUseLine: true,
		//TraverseChildren:      true,
	})
	cmd.AddCommand(
		build.Command(),
		list.Command(),
		pull.Command(),
		push.Command(),
		remove.Command(),
		tag.Command(),
	)
	return cmd
}

type CommandSpec struct {
}

func (s *CommandSpec) Run(cmd *cobra.Command, _ []string) error {
	return cmd.Help()
}
