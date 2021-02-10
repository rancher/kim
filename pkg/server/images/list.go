package images

import (
	"context"

	imagesv1 "github.com/rancher/kim/pkg/apis/services/images/v1alpha1"
	criv1 "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
)

// List images server-side impl
func (s *Server) List(ctx context.Context, req *imagesv1.ImageListRequest) (*imagesv1.ImageListResponse, error) {
	res, err := s.ImageService().ListImages(ctx, &criv1.ListImagesRequest{Filter: req.Filter})
	if err != nil {
		return nil, err
	}
	return &imagesv1.ImageListResponse{
		Images: res.Images,
	}, nil
}
