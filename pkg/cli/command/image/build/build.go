package build

import (
	"os"

	"github.com/rancher/kim/pkg/client"
	"github.com/rancher/kim/pkg/client/image"
	wrangler "github.com/rancher/wrangler-cli"
	"github.com/spf13/cobra"
)

const (
	Use   = "build [OPTIONS] PATH"
	Short = "Build an image"
)

func Command() *cobra.Command {
	return wrangler.Command(&CommandSpec{}, cobra.Command{
		Use:                   Use,
		Short:                 Short,
		DisableFlagsInUseLine: true,
		Args:                  cobra.ExactArgs(1),
	})
}

type CommandSpec struct {
	image.Build
}

func (c *CommandSpec) Run(cmd *cobra.Command, args []string) error {
	k8s, err := client.DefaultConfig.Interface()
	if err != nil {
		return err
	}
	path := args[0]
	if path == "" || path == "." {
		path, err = os.Getwd()
	}
	if err != nil {
		return err
	}
	return c.Build.Do(cmd.Context(), k8s, path)
}
