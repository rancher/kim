package provider

import (
	"fmt"
	"os/exec"

	"github.com/docker/distribution/reference"
	"github.com/docker/docker-credential-helpers/client"
	"k8s.io/kubernetes/pkg/credentialprovider"
)

func RegisterDockerCredentialHelper(name string) {
	credentialprovider.RegisterCredentialProvider(name, &helperConfigProvider{helper: name})
}

type helperConfigProvider struct {
	helper string
}

func (p *helperConfigProvider) Enabled() bool {
	_, err := exec.LookPath(fmt.Sprintf("docker-credential-%s", p.helper))
	return err == nil
}

func (p *helperConfigProvider) Provide(image string) credentialprovider.DockerConfig {
	helper := client.NewShellProgramFunc(fmt.Sprintf("docker-credential-%s", p.helper))
	config := credentialprovider.DockerConfig{}
	repository := imageToRepositoryURL(image)
	credentials, err := client.Get(helper, repository)
	if err == nil && credentials != nil {
		config[image] = credentialprovider.DockerConfigEntry{
			Username: credentials.Username,
			Password: credentials.Secret,
			Provider: p,
		}
	}
	return config
}

const defaultRepositoryURL = "https://index.docker.io/v1/"

func imageToRepositoryURL(image string) string {
	name, err := reference.ParseNormalizedNamed(image)
	if err != nil {
		return defaultRepositoryURL
	}
	domain := reference.Domain(name)
	if domain == "docker.io" {
		return defaultRepositoryURL
	}
	return fmt.Sprintf("https://%s", domain)
}
