package install

import (
	"context"

	"github.com/rancher/kim/pkg/client"
	"github.com/rancher/kim/pkg/client/builder"
	wrangler "github.com/rancher/wrangler-cli"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Command() *cobra.Command {
	cmd := wrangler.Command(&CommandSpec{}, cobra.Command{
		Use:                   "install [OPTIONS]",
		Short:                 "Install builder component(s)",
		DisableFlagsInUseLine: true,
	})
	cmd.Flag("endpoint-addr").Hidden = true // because the "hidden" annotation is not supported
	return cmd
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

func Check(ctx context.Context) error {
	pre := CommandSpec{}
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
	return pre.Install.Do(ctx, k8s)
}
