package remove

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/rancher/kim/pkg/client"
	"github.com/rancher/kim/pkg/client/image"
	wrangler "github.com/rancher/wrangler-cli"
	"github.com/spf13/cobra"
)

const (
	Short = "Remove an image"
)

func Use(sub string) string {
	return fmt.Sprintf("%s [OPTIONS] IMAGE [IMAGE...]", sub)
}

func Command() *cobra.Command {
	return wrangler.Command(&CommandSpec{}, cobra.Command{
		Use:                   Use("rm"),
		Short:                 Short,
		Aliases:               []string{"remove"},
		DisableFlagsInUseLine: true,
	})
}

type CommandSpec struct {
	image.Remove
}

func (c *CommandSpec) Run(cmd *cobra.Command, args []string) error {
	if len(args) != 1 {
		return errors.New("exactly one argument is required")
	}

	k8s, err := client.DefaultConfig.Interface()
	if err != nil {
		return err
	}
	return c.Remove.Do(cmd.Context(), k8s, args[0])
}
