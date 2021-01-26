package cli

import (
	"github.com/rancher/kim/pkg/credential/provider"
	"github.com/spf13/cobra"
)

func (s *App) persistentPre(_ *cobra.Command, _ []string) error {
	provider.RegisterDockerCredentialHelper("acr-darwin")
	provider.RegisterDockerCredentialHelper("desktop")
	provider.RegisterDockerCredentialHelper("osxkeychain")
	return nil
}
