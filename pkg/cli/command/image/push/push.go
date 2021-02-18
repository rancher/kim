package push

import (
	"github.com/rancher/kim/pkg/client"
	"github.com/rancher/kim/pkg/client/image"
	wrangler "github.com/rancher/wrangler-cli"
	"github.com/spf13/cobra"
)

const (
	Use   = "push [OPTIONS] IMAGE"
	Short = "Push an image"
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
	image.Push
}

func (s *CommandSpec) Run(cmd *cobra.Command, args []string) error {
	k8s, err := client.DefaultConfig.Interface()
	if err != nil {
		return err
	}
	return s.Push.Do(cmd.Context(), k8s, args[0])
}
