package images

import (
	"context"

	imagesv1 "github.com/rancher/kim/pkg/apis/services/images/v1alpha1"
	criv1 "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
)

// Remove image server-side impl
func (s *Server) Remove(ctx context.Context, req *imagesv1.ImageRemoveRequest) (*imagesv1.ImageRemoveResponse, error) {
	_, err := s.ImageService().RemoveImage(ctx, &criv1.RemoveImageRequest{Image: req.Image})
	if err != nil {
		return nil, err
	}
	return &imagesv1.ImageRemoveResponse{}, nil
}
