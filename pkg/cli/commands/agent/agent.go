package agent

import (
	"github.com/rancher/kim/pkg/server"
	"github.com/rancher/kim/pkg/server/action"
	wrangler "github.com/rancher/wrangler-cli"
	"github.com/spf13/cobra"
)

func Command() *cobra.Command {
	return wrangler.Command(&CommandSpec{}, cobra.Command{
		Use:                   "agent [OPTIONS]",
		Short:                 "Run the controller daemon",
		Hidden:                true,
		DisableFlagsInUseLine: true,
	})
}

type CommandSpec struct {
	action.Agent
}

func (s *CommandSpec) Customize(cmd *cobra.Command) {
	d := cmd.Flag("agent-image")
	d.DefValue = server.DefaultAgentImage
	cmd.Flags().AddFlag(d)
}

func (s *CommandSpec) Run(cmd *cobra.Command, args []string) error {
	return s.Agent.Run(cmd.Context())
}
