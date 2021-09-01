package pull

import (
	"github.com/rancher/kim/pkg/cli/command/builder/install"
	"github.com/rancher/kim/pkg/client"
	"github.com/rancher/kim/pkg/client/image"
	wrangler "github.com/rancher/wrangler-cli"
	"github.com/spf13/cobra"
)

const (
	Use   = "pull [OPTIONS] IMAGE"
	Short = "Pull an image"
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
	image.Pull
}

func (s *CommandSpec) Run(cmd *cobra.Command, args []string) error {
	k8s, err := client.DefaultConfig.Interface()
	if err != nil {
		return err
	}
	err = install.Check(cmd.Context())
	if err != nil {
		return err
	}
	return s.Pull.Do(cmd.Context(), k8s, args[0])
}
