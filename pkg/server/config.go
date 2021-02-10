package server

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/containerd/containerd"
	"github.com/docker/distribution/reference"
	buildkit "github.com/moby/buildkit/client"
	"github.com/pkg/errors"
	"github.com/rancher/kim/pkg/client"
	"github.com/rancher/kim/pkg/server/images"
	"github.com/rancher/kim/pkg/version"
	"google.golang.org/grpc"
)

const (
	defaultAgentPort     = 1233
	defaultAgentImage    = "docker.io/rancher/kim"
	defaultBuildkitImage = "docker.io/moby/buildkit:v0.8.1"
)

var (
	DefaultAgentPort     = defaultAgentPort
	DefaultAgentImage    = defaultAgentImage
	DefaultBuildkitImage = defaultBuildkitImage
)

type Config struct {
	AgentImage        string `usage:"Image to run the agent w/ missing tag inferred from version"`
	AgentPort         int    `usage:"Port that the agent will listen on" default:"1233"`
	BuildkitImage     string `usage:"BuildKit image for running buildkitd" default:"docker.io/moby/buildkit:v0.8.1"`
	BuildkitNamespace string `usage:"BuildKit namespace in containerd (not 'k8s.io')" default:"buildkit"`
	BuildkitPort      int    `usage:"BuildKit service port" default:"1234"`
	BuildkitSocket    string `usage:"BuildKit socket address" default:"unix:///run/buildkit/buildkitd.sock"`
	ContainerdSocket  string `usage:"Containerd socket address" default:"/run/k3s/containerd/containerd.sock"`
}

func (c *Config) GetAgentImage() (string, error) {
	if c.AgentImage == "" {
		c.AgentImage = DefaultAgentImage
	}
	if c.AgentImage == DefaultAgentImage {
		ref, err := reference.ParseAnyReference(c.AgentImage)
		if err != nil {
			return c.AgentImage, errors.Wrap(err, "failed to parse agent image")
		}
		if named, ok := ref.(reference.Named); ok {
			if reference.IsNameOnly(named) {
				tagged, err := reference.WithTag(named, strings.ReplaceAll(version.Version, "+", "-"))
				if err != nil {
					return c.AgentImage, errors.Wrap(err, "failed to append version tag")
				}
				return reference.FamiliarString(tagged), nil
			}
		}
	}
	return c.AgentImage, nil
}

func (c *Config) GetBuildkitImage() (string, error) {
	if c.BuildkitImage == "" {
		c.BuildkitImage = DefaultBuildkitImage
	}
	return c.BuildkitImage, nil
}

func (c *Config) Interface(ctx context.Context, config *client.Config) (*images.Server, error) {
	k8s, err := config.Interface()
	if err != nil {
		return nil, err
	}
	server := images.Server{
		Kubernetes: k8s,
	}

	server.Buildkit, err = buildkit.New(ctx, c.BuildkitSocket)
	if err != nil {
		return nil, err
	}
	conn, err := grpc.DialContext(ctx, fmt.Sprintf("unix://%s", c.ContainerdSocket), grpc.WithInsecure(), grpc.WithBlock(), grpc.FailOnNonTempDialError(true))
	if err != nil {
		server.Close()
		return nil, err
	}
	server.Containerd, err = containerd.NewWithConn(conn,
		containerd.WithDefaultNamespace(c.BuildkitNamespace),
		containerd.WithTimeout(5*time.Second),
	)
	if err != nil {
		server.Close()
		return nil, err
	}

	return &server, nil
}
