package images

import (
	"context"

	imagesv1 "github.com/rancher/kim/pkg/apis/services/images/v1alpha1"
	criv1 "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
)

// Status of an image server-side impl (unused)
func (s *Server) Status(ctx context.Context, req *imagesv1.ImageStatusRequest) (*imagesv1.ImageStatusResponse, error) {
	res, err := s.ImageService().ImageStatus(ctx, &criv1.ImageStatusRequest{Image: req.Image})
	if err != nil {
		return nil, err
	}
	return &imagesv1.ImageStatusResponse{
		Image: res.Image,
	}, nil
}
