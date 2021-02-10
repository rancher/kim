package images

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/remotes"
	"github.com/containerd/containerd/remotes/docker"
	buildkit "github.com/moby/buildkit/client"
	imagesv1 "github.com/rancher/kim/pkg/apis/services/images/v1alpha1"
	"github.com/rancher/kim/pkg/auth"
	"github.com/rancher/kim/pkg/client"
	"github.com/rancher/kim/pkg/version"
	"github.com/sirupsen/logrus"
	criv1 "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
)

var _ imagesv1.ImagesServer = &Server{}

type Server struct {
	Kubernetes *client.Interface
	Buildkit   *buildkit.Client
	Containerd *containerd.Client

	criImages criv1.ImageServiceClient
	criOnce   sync.Once

	pushJobs sync.Map
}

func (s *Server) ImageService() criv1.ImageServiceClient {
	s.criOnce.Do(func() {
		s.criImages = criv1.NewImageServiceClient(s.Containerd.Conn())
	})
	return s.criImages
}

// Close the Server connections to various backends.
func (s *Server) Close() {
	if s.Buildkit != nil {
		if err := s.Buildkit.Close(); err != nil {
			logrus.Warnf("error closing connection to buildkit: %v", err)
		}
	}
	if s.Containerd != nil {
		// this will close the underlying grpc connection making the cri runtime/images clients inoperable as well
		if err := s.Containerd.Close(); err != nil {
			logrus.Warnf("error closing connection to containerd: %v", err)
		}
	}
}

func Resolver(authConfig *criv1.AuthConfig, statusTracker docker.StatusTracker) remotes.Resolver {
	authorizer := docker.NewDockerAuthorizer(
		docker.WithAuthClient(http.DefaultClient),
		docker.WithAuthCreds(func(host string) (string, string, error) {
			return auth.Parse(authConfig, host)
		}),
		docker.WithAuthHeader(http.Header{
			"User-Agent": []string{fmt.Sprintf("rancher-kim/%s", version.Version)},
		}),
	)
	return docker.NewResolver(docker.ResolverOptions{
		Tracker: statusTracker,
		Hosts: docker.ConfigureDefaultRegistries(
			docker.WithAuthorizer(authorizer),
		),
	})

}
