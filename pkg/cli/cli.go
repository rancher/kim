package cli

import (
	"fmt"
	"strings"

	"github.com/rancher/kim/pkg/cli/command/agent"
	"github.com/rancher/kim/pkg/cli/command/builder"
	"github.com/rancher/kim/pkg/cli/command/image"
	"github.com/rancher/kim/pkg/cli/command/image/build"
	"github.com/rancher/kim/pkg/cli/command/image/list"
	"github.com/rancher/kim/pkg/cli/command/image/pull"
	"github.com/rancher/kim/pkg/cli/command/image/push"
	"github.com/rancher/kim/pkg/cli/command/image/remove"
	"github.com/rancher/kim/pkg/cli/command/image/tag"
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

Available Commands:{{range .Commands}}{{if (and (not (index .Annotations "shortcut-root")) (or .IsAvailableCommand (eq .Name "help")))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}{{if (eq (index .Annotations "shortcuts") "image")}}

Images Shortcuts:{{range .Commands}}{{if (and .IsAvailableCommand (eq (index .Annotations "shortcut-root") "image"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

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
	app := wrangler.Command(&App{}, cobra.Command{
		Use:                   "kim [OPTIONS] COMMAND",
		Short:                 "Kubernetes Image Manager -- in ur kubernetes buildin ur imagez",
		Version:               version.FriendlyVersion(),
		Example:               "kim image build --tag your/image:tag .",
		DisableFlagsInUseLine: true,
		Annotations: map[string]string{
			"shortcuts": "image",
		},
	})
	app.SetUsageTemplate(defaultUsageTemplate)
	app.AddCommand(
		agent.Command(),
		image.Command(),
		builder.Command(),
	)
	// image subsystem shortcuts
	AddShortcut(app, build.Use, "image", "build")
	AddShortcut(app, list.Use("images"), "image", "list")
	AddShortcut(app, pull.Use, "image", "pull")
	AddShortcut(app, push.Use, "image", "push")
	AddShortcut(app, remove.Use("rmi"), "image", "remove")
	AddShortcut(app, tag.Use, "image", "tag")
	return app
}

func Image(exe string) *cobra.Command {
	cmd := image.Command()
	app := wrangler.Command(&App{}, *cmd)
	app.SetUsageTemplate(defaultUsageTemplate)
	app.Use = image.Use(exe)
	app.Version = version.FriendlyVersion()
	app.Example = fmt.Sprintf("%s build --tag your/image:tag .", strings.Replace(exe, "kubectl-", "kubectl ", 1))
	cmd.PersistentFlags().AddFlagSet(app.PersistentFlags())
	return app
}

func Builder(exe string) *cobra.Command {
	cmd := builder.Command()
	app := wrangler.Command(&App{}, *cmd)
	app.SetUsageTemplate(defaultUsageTemplate)
	app.Use = builder.Use(exe)
	app.Version = version.FriendlyVersion()
	app.Example = fmt.Sprintf("%s install --selector k3s.io/hostname=my-builder", strings.Replace(exe, "kubectl-", "kubectl ", 1))
	cmd.PersistentFlags().AddFlagSet(app.PersistentFlags())
	return app
}

func AddShortcut(cmd *cobra.Command, use string, path ...string) {
	sub, _, err := cmd.Find(path)
	if err != nil {
		panic(err)
	}
	target := strings.Join(path, " ")
	shortcut := *sub
	shortcut.Use = use
	//shortcut.Short = fmt.Sprintf("%s (shortcut to `%s %s`)", sub.Short, cmd.Name(), target)
	shortcut.Aliases = []string{target}
	shortcut.Annotations = map[string]string{
		"shortcut-root": path[0],
	}
	for pre := sub; pre != nil; pre = pre.Parent() {
		if pre.Name() == path[0] {
			if pre.PersistentPreRunE != nil {
				shortcut.PersistentPreRunE = func(alias *cobra.Command, args []string) error {
					return pre.PersistentPreRunE(alias, args)
				}
			}
			break
		}
	}
	cmd.AddCommand(&shortcut)
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
