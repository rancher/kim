package images

import (
	"context"
	"time"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/images"
	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/containerd/remotes"
	"github.com/containerd/containerd/remotes/docker"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	imagesv1 "github.com/rancher/kim/pkg/apis/services/images/v1alpha1"
	"github.com/rancher/kim/pkg/progress"
	"github.com/sirupsen/logrus"
)

var (
	PushTracker = docker.NewInMemoryTracker()
)

// Push server-side impl
func (s *Server) Push(ctx context.Context, req *imagesv1.ImagePushRequest) (*imagesv1.ImagePushResponse, error) {
	ctx = namespaces.WithNamespace(ctx, "k8s.io")
	img, err := s.Containerd.ImageService().Get(ctx, req.Image.Image)
	if err != nil {
		return nil, err
	}

	resolver := Resolver(req.Auth, PushTracker)
	tracker := progress.NewTracker(ctx, PushTracker)
	s.pushJobs.Store(img.Name, tracker)
	handler := images.HandlerFunc(func(ctx context.Context, desc ocispec.Descriptor) ([]ocispec.Descriptor, error) {
		tracker.Add(remotes.MakeRefKey(ctx, desc))
		return nil, nil
	})
	err = s.Containerd.Push(ctx, img.Name, img.Target,
		containerd.WithResolver(resolver),
		containerd.WithImageHandler(handler),
	)
	if err != nil {
		return nil, err
	}
	return &imagesv1.ImagePushResponse{
		Image: img.Name,
	}, nil
}

// PushProgress server-side impl
func (s *Server) PushProgress(req *imagesv1.ImageProgressRequest, srv imagesv1.Images_PushProgressServer) error {
	ctx := namespaces.WithNamespace(srv.Context(), "k8s.io")
	defer s.pushJobs.Delete(req.Image)

	timeout := time.After(15 * time.Second)

	for {
		if tracker, tracking := s.pushJobs.Load(req.Image); tracking {
			for status := range tracker.(progress.Tracker).Status() {
				if err := srv.Send(&imagesv1.ImageProgressResponse{Status: status}); err != nil {
					logrus.Debugf("push-progress-error: %s -> %v", req.Image, err)
					return err
				}
			}
			logrus.Debugf("push-progress-done: %s", req.Image)
			return nil
		}
		select {
		case <-timeout:
			logrus.Debugf("push-progress-timeout: not tracking %s", req.Image)
			return nil
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
	}
}
