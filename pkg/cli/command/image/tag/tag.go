package tag

import (
	"github.com/rancher/kim/pkg/client"
	"github.com/rancher/kim/pkg/client/image"
	wrangler "github.com/rancher/wrangler-cli"
	"github.com/spf13/cobra"
)

const (
	Use   = "tag SOURCE_REF TARGET_REF [TARGET_REF, ...]"
	Short = "Tag an image"
)

func Command() *cobra.Command {
	return wrangler.Command(&CommandSpec{}, cobra.Command{
		Use:                   Use,
		Short:                 Short,
		DisableFlagsInUseLine: true,
		Args:                  cobra.MinimumNArgs(2),
	})
}

type CommandSpec struct {
	image.Tag
}

func (c *CommandSpec) Run(cmd *cobra.Command, args []string) error {
	k8s, err := client.DefaultConfig.Interface()
	if err != nil {
		return err
	}
	return c.Tag.Do(cmd.Context(), k8s, args[0], args[1:])
}
