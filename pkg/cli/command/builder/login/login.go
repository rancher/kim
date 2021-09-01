package login

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"strings"

	"github.com/moby/term"
	"github.com/pkg/errors"
	"github.com/rancher/kim/pkg/client"
	"github.com/rancher/kim/pkg/client/builder"
	wrangler "github.com/rancher/wrangler-cli"
	"github.com/spf13/cobra"
	"k8s.io/kubernetes/pkg/credentialprovider"
)

func Command() *cobra.Command {
	return wrangler.Command(&CommandSpec{}, cobra.Command{
		Use:                   "login [OPTIONS] [SERVER]",
		Short:                 "Establish credentials for a registry.",
		DisableFlagsInUseLine: true,
		Args:                  cobra.ExactArgs(1),
	})
}

type CommandSpec struct {
	builder.Login
}

func (s *CommandSpec) Run(cmd *cobra.Command, args []string) error {
	k8s, err := client.DefaultConfig.Interface()
	if err != nil {
		return err
	}
	if s.PasswordStdin {
		if s.Password != "" {
			return errors.New("--password and --password-stdin are mutually exclusive")
		}
		if s.Username == "" {
			return errors.New("must provide --username with --password-stdin")
		}
		password, err := ioutil.ReadAll(cmd.InOrStdin())
		if err != nil {
			return err
		}
		s.Password = strings.TrimSuffix(string(password), "\n")
		s.Password = strings.TrimSuffix(s.Password, "\r")
	}
	if (s.Username == "" || s.Password == "") && !term.IsTerminal(os.Stdout.Fd()) {
		return errors.New("cannot perform interactive login from non tty device")
	}
	if s.Username == "" {
		fmt.Fprintf(os.Stdout, "Username: ")
		reader := bufio.NewReader(os.Stdin)
		line, _, err := reader.ReadLine()
		if err != nil {
			return err
		}
		s.Username = strings.TrimSpace(string(line))
	}
	if s.Password == "" {
		state, err := term.SaveState(os.Stdin.Fd())
		if err != nil {
			return err
		}
		fmt.Fprintf(os.Stdout, "Password: ")
		term.DisableEcho(os.Stdin.Fd(), state)
		reader := bufio.NewReader(os.Stdin)
		line, _, err := reader.ReadLine()
		if err != nil {
			return err
		}
		fmt.Fprintln(os.Stdout)
		term.RestoreTerminal(os.Stdin.Fd(), state)
		s.Password = strings.TrimSpace(string(line))
		if s.Password == "" {
			return errors.New("password is required")
		}
	}
	server, err := credentialprovider.ParseSchemelessURL(args[0])
	if err != nil {
		if server, err = url.Parse(args[0]); err != nil {
			return err
		}
	}
	// special case for [*.]docker.io -> https://index.docker.io/v1/
	if strings.HasSuffix(server.Host, "docker.io") {
		server.Scheme = "https"
		server.Host = "index.docker.io"
		if server.Path == "" {
			server.Path = "/v1/"
		}
		return s.Login.Do(cmd.Context(), k8s, server.String())
	}
	return s.Login.Do(cmd.Context(), k8s, server.Host)
}
