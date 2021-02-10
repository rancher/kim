package images

import (
	"sync"

	criv1 "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"

	"github.com/containerd/containerd"
	buildkit "github.com/moby/buildkit/client"
	imagesv1 "github.com/rancher/kim/pkg/apis/services/images/v1alpha1"
	"github.com/rancher/kim/pkg/client"
	"github.com/sirupsen/logrus"
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
