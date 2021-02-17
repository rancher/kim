package install

import (
	"github.com/rancher/kim/pkg/client"
	"github.com/rancher/kim/pkg/client/builder"
	wrangler "github.com/rancher/wrangler-cli"
	"github.com/spf13/cobra"
)

func Command() *cobra.Command {
	return wrangler.Command(&CommandSpec{}, cobra.Command{
		Use:                   "install [OPTIONS]",
		Short:                 "Install builder component(s)",
		DisableFlagsInUseLine: true,
	})
}

type CommandSpec struct {
	builder.Install
}

func (s *CommandSpec) Run(cmd *cobra.Command, _ []string) error {
	k8s, err := client.DefaultConfig.Interface()
	if err != nil {
		return err
	}
	return s.Install.Do(cmd.Context(), k8s)
}
