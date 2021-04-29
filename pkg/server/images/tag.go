package images

import (
	"context"

	"github.com/containerd/containerd/errdefs"
	"github.com/containerd/containerd/namespaces"
	imagesv1 "github.com/rancher/kim/pkg/apis/services/images/v1alpha1"
	"github.com/sirupsen/logrus"
	criv1 "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
)

// Tag image server-side impl, adapted from containerd's `ctr tag` implementation
func (s *Server) Tag(ctx context.Context, req *imagesv1.ImageTagRequest) (*imagesv1.ImageTagResponse, error) {
	// containerd services require a namespace
	ctx, done, err := s.Containerd.WithLease(namespaces.WithNamespace(ctx, "k8s.io"))
	if err != nil {
		return nil, err
	}
	defer done(ctx)
	ref := req.Image.Image
	svc := s.Containerd.ImageService()
	img, err := svc.Get(ctx, ref)
	if err != nil {
		return nil, err
	}
	for _, tag := range req.Tags {
		img.Name = tag
		// Attempt to create the image first
		if _, err = svc.Create(ctx, img); err != nil {
			if errdefs.IsAlreadyExists(err) {
				if err = svc.Delete(ctx, tag); err != nil {
					return nil, err
				}
				if _, err = svc.Create(ctx, img); err != nil {
					return nil, err
				}
			} else {
				return nil, err
			}
		}
		logrus.Debugf("image-tag: %#v", img)
	}
	res, err := s.ImageService().ImageStatus(ctx, &criv1.ImageStatusRequest{Image: req.Image})
	if err != nil {
		return nil, err
	}
	return &imagesv1.ImageTagResponse{
		Image: res.Image,
	}, nil
}
