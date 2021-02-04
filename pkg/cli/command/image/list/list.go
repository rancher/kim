package list

import (
	"fmt"

	"github.com/rancher/kim/pkg/client"
	"github.com/rancher/kim/pkg/client/image"
	wrangler "github.com/rancher/wrangler-cli"
	"github.com/spf13/cobra"
)

const (
	Short = "List images"
)

func Use(sub string) string {
	return fmt.Sprintf("%s [OPTIONS] [REPOSITORY[:TAG]]", sub)
}

func Command() *cobra.Command {
	return wrangler.Command(&CommandSpec{}, cobra.Command{
		Use:                   Use("ls"),
		Short:                 Short,
		DisableFlagsInUseLine: true,
		Aliases:               []string{"list"},
	})
}

type CommandSpec struct {
	image.List
}

func (s *CommandSpec) Run(cmd *cobra.Command, args []string) error {
	k8s, err := client.DefaultConfig.Interface()
	if err != nil {
		return err
	}

	return s.List.Do(cmd.Context(), k8s, args)
}
