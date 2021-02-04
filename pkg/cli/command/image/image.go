package image

import (
	"github.com/rancher/kim/pkg/cli/command/image/build"
	"github.com/rancher/kim/pkg/cli/command/image/list"
	"github.com/rancher/kim/pkg/cli/command/image/pull"
	"github.com/rancher/kim/pkg/cli/command/image/push"
	"github.com/rancher/kim/pkg/cli/command/image/remove"
	"github.com/rancher/kim/pkg/cli/command/image/tag"
	"github.com/rancher/kim/pkg/cli/command/system/install"
	"github.com/rancher/kim/pkg/client"
	wrangler "github.com/rancher/wrangler-cli"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Command() *cobra.Command {
	cmd := wrangler.Command(&CommandSpec{}, cobra.Command{
		Use:                   "image [OPTIONS] COMMAND",
		Short:                 "Manage Images",
		DisableFlagsInUseLine: true,
		//TraverseChildren:      true,
	})
	cmd.AddCommand(
		build.Command(),
		list.Command(),
		pull.Command(),
		push.Command(),
		remove.Command(),
		tag.Command(),
	)
	return cmd
}

type CommandSpec struct {
}

func (s *CommandSpec) PersistentPre(cmd *cobra.Command, _ []string) error {
	pre := install.CommandSpec{}
	// i've tried using subcommands from the cli command tree but there be dragons
	wrangler.Command(&pre, cobra.Command{}) // initialize pre.Install defaults
	k8s, err := client.DefaultConfig.Interface()
	if err != nil {
		return err
	}
	// if the daemon-set is available then we don't need to do anything
	daemon, err := k8s.Apps.DaemonSet().Get(k8s.Namespace, "builder", metav1.GetOptions{})
	if err == nil && daemon.Status.NumberAvailable > 0 {
		return nil
	}
	pre.NoWait = false
	pre.NoFail = true
	logrus.Warnf("Cannot find available builder daemon, attempting automatic installation...")
	return pre.Install.Do(cmd.Context(), k8s)
}

func (s *CommandSpec) Run(cmd *cobra.Command, _ []string) error {
	return cmd.Help()
}
