package uninstall

import (
	"github.com/rancher/kim/pkg/client"
	"github.com/rancher/kim/pkg/client/system/builder"
	wrangler "github.com/rancher/wrangler-cli"
	"github.com/spf13/cobra"
)

func Command() *cobra.Command {
	return wrangler.Command(&CommandSpec{}, cobra.Command{
		Use:                   "uninstall [OPTIONS]",
		Short:                 "Uninstall builder component(s)",
		DisableFlagsInUseLine: true,
	})
}

type CommandSpec struct {
	builder.Uninstall
}

func (s *CommandSpec) Run(cmd *cobra.Command, args []string) error {
	k8s, err := client.DefaultConfig.Interface()
	if err != nil {
		return err
	}
	ctx := cmd.Context()
	err = s.Uninstall.Namespace(ctx, k8s)
	if err != nil {
		return err
	}
	return s.Uninstall.NodeRole(ctx, k8s)
}
