package cli

import (
	"github.com/rancher/kim/pkg/cli/commands/agent"
	"github.com/rancher/kim/pkg/cli/commands/build"
	"github.com/rancher/kim/pkg/cli/commands/images"
	"github.com/rancher/kim/pkg/cli/commands/info"
	"github.com/rancher/kim/pkg/cli/commands/install"
	"github.com/rancher/kim/pkg/cli/commands/pull"
	"github.com/rancher/kim/pkg/cli/commands/push"
	"github.com/rancher/kim/pkg/cli/commands/rmi"
	"github.com/rancher/kim/pkg/cli/commands/tag"
	"github.com/rancher/kim/pkg/cli/commands/uninstall"
	"github.com/rancher/kim/pkg/client"
	"github.com/rancher/kim/pkg/credential/provider"
	"github.com/rancher/kim/pkg/version"
	wrangler "github.com/rancher/wrangler-cli"
	"github.com/spf13/cobra"
)

const (
	// a very slight customization of the spf13/cobra default usage template (example is indented)
	defaultUsageTemplate = `Usage:{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command]{{end}}{{if gt (len .Aliases) 0}}

Aliases:
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

Examples:
  {{.Example}}{{end}}{{if .HasAvailableSubCommands}}

Available Commands:{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Global Flags:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`
)

var (
	DebugConfig wrangler.DebugConfig
)

func Main() *cobra.Command {
	cmd := wrangler.Command(&App{}, cobra.Command{
		Use:                   "kim [OPTIONS] COMMAND",
		Short:                 "Kubernetes Image Manager -- in ur kubernetes buildin ur imagez",
		Version:               version.FriendlyVersion(),
		Example:               "kim build --tag your/image:tag .",
		DisableFlagsInUseLine: true,
	})
	cmd.AddCommand(
		agent.Command(),
		info.Command(),
		images.Command(),
		install.Command(),
		uninstall.Command(),
		build.Command(),
		pull.Command(),
		push.Command(),
		rmi.Command(),
		tag.Command(),
	)
	cmd.SetUsageTemplate(defaultUsageTemplate)
	return cmd
}

type App struct {
	wrangler.DebugConfig
	client.Config
}

func (s *App) Customize(cmd *cobra.Command) {
	d := cmd.Flag("namespace")
	if d.DefValue == "" || d.DefValue == "default" {
		d.DefValue = client.DefaultNamespace
	}
	cmd.Flags().AddFlag(d)
}

func (s *App) Run(cmd *cobra.Command, _ []string) error {
	return cmd.Help()
}

func (s *App) PersistentPre(cmd *cobra.Command, args []string) error {
	s.MustSetupDebug()
	DebugConfig = s.DebugConfig
	client.DefaultConfig = s.Config

	provider.RegisterDockerCredentialHelper("gcloud")
	provider.RegisterDockerCredentialHelper("pass")
	return s.persistentPre(cmd, args)
}
