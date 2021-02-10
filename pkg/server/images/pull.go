package images

import (
	"context"
	"fmt"
	"time"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/containerd/platforms"
	imagesv1 "github.com/rancher/kim/pkg/apis/services/images/v1alpha1"
	"github.com/rancher/kim/pkg/version"
	"github.com/sirupsen/logrus"
	criv1 "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
)

// Pull server-side impl
func (s *Server) Pull(ctx context.Context, req *imagesv1.ImagePullRequest) (*imagesv1.ImagePullResponse, error) {
	logrus.Debugf("image-pull: %#v", req)
	var err error
	if req.Image.Annotations != nil && req.Image.Annotations["images.cattle.io/pull-backend"] == "cri" {
		err = s.pullCRI(ctx, req.Image, req.Auth)
	} else {
		err = s.pullCTD(ctx, req.Image, req.Auth)
	}
	if err != nil {
		return nil, err
	}
	return &imagesv1.ImagePullResponse{
		Image: req.Image.Image,
	}, nil
}

// pullCTD attempts to pull via containerd directly
func (s *Server) pullCTD(ctx context.Context, image *criv1.ImageSpec, auth *criv1.AuthConfig) error {
	ctx = namespaces.WithNamespace(ctx, "k8s.io")
	resolver := Resolver(auth, nil)
	platform := platforms.DefaultString()
	if image.Annotations != nil {
		platform = image.Annotations["images.cattle.io/pull-platform"]
	}
	_, err := s.Containerd.Pull(ctx, image.Image,
		containerd.WithPullUnpack,
		containerd.WithSchema1Conversion,
		containerd.WithPullLabel("io.cattle.images/client", fmt.Sprintf("kim/%s", version.Version)),
		containerd.WithResolver(resolver),
		containerd.WithPlatform(platform),
	)
	return err
}

// pullCRI attempts to pull via CRI
func (s *Server) pullCRI(ctx context.Context, image *criv1.ImageSpec, auth *criv1.AuthConfig) error {
	logrus.Debugf("image-pull-cri: %#v", image)
	_, err := s.ImageService().PullImage(ctx, &criv1.PullImageRequest{
		Auth:  auth,
		Image: image,
	})
	return err
}

// PullProgress server-side impl
func (s *Server) PullProgress(req *imagesv1.ImageProgressRequest, srv imagesv1.Images_PullProgressServer) error {
	logrus.Debugf("image-pull-progress: %#v", req)
	ctx := namespaces.WithNamespace(srv.Context(), "k8s.io")

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
			isr, err := s.ImageService().ImageStatus(ctx, &criv1.ImageStatusRequest{
				Image: &criv1.ImageSpec{
					Image: req.Image,
				},
			})
			if err != nil {
				logrus.Debugf("pull-progress-image-status-error: %v", err)
				return err
			}
			if isr.Image != nil {
				logrus.Debugf("pull-progress-image-status-done: %s", isr.Image)
				return nil
			}
			csl, err := s.Containerd.ContentStore().ListStatuses(ctx, "") // TODO is this filter too broad?
			if err != nil {
				logrus.Debugf("pull-progress-content-status-error: %v", err)
				return err
			}
			res := &imagesv1.ImageProgressResponse{}
			for _, s := range csl {
				status := "waiting"
				if s.Offset == s.Total {
					status = "unpacking"
				} else if s.Offset > 0 {
					status = "downloading"
				}
				res.Status = append(res.Status, imagesv1.ImageStatus{
					Status:    status,
					Ref:       s.Ref,
					Offset:    s.Offset,
					Total:     s.Total,
					StartedAt: s.StartedAt,
					UpdatedAt: s.UpdatedAt,
				})
			}
			if err = srv.Send(res); err != nil {
				logrus.Debugf("pull-progress-content-send-error: %v", err)
				return err
			}
		}
	}
}
