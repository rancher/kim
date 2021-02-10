package images

import (
	"context"

	"github.com/containerd/containerd/errdefs"
	"github.com/containerd/containerd/images"
	"github.com/containerd/containerd/namespaces"
	imagesv1 "github.com/rancher/kim/pkg/apis/services/images/v1alpha1"
	"github.com/sirupsen/logrus"
	criv1 "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
)

// Remove image server-side impl
func (s *Server) Remove(ctx context.Context, req *imagesv1.ImageRemoveRequest) (*imagesv1.ImageRemoveResponse, error) {
	logrus.Debugf("image-remove: %#v", req.Image)
	ctx, done, err := s.Containerd.WithLease(namespaces.WithNamespace(ctx, "k8s.io"))
	if err != nil {
		return nil, err
	}
	defer done(ctx)
	err = s.Containerd.ImageService().Delete(ctx, req.Image.Image, images.SynchronousDelete())
	if errdefs.IsNotFound(err) {
		// at this point we assume it is an image id and fallback to cri behavior, aka remove every tag/digest
		_, err = s.ImageService().RemoveImage(ctx, &criv1.RemoveImageRequest{Image: req.Image})
	}
	if err != nil {
		return nil, err
	}
	return &imagesv1.ImageRemoveResponse{}, nil
}
