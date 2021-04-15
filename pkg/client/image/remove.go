package image

import (
	"context"

	"github.com/docker/distribution/reference"
	imagesv1 "github.com/rancher/kim/pkg/apis/services/images/v1alpha1"
	"github.com/rancher/kim/pkg/client"
	"github.com/sirupsen/logrus"
	criv1 "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
)

type Remove struct {
}

func (s *Remove) Do(ctx context.Context, k8s *client.Interface, image string) error {
	if named, err := reference.ParseNormalizedNamed(image); err == nil {
		image = reference.TagNameOnly(named).String()
	}
	return client.Images(ctx, k8s, func(ctx context.Context, imagesClient imagesv1.ImagesClient) error {
		req := &imagesv1.ImageRemoveRequest{
			Image: &criv1.ImageSpec{
				Image: image,
			},
		}
		if ref, err := reference.ParseAnyReference(image); err == nil {
			statusRequest := imagesv1.ImageStatusRequest{
				Image: &criv1.ImageSpec{
					Image: ref.String(),
				},
			}
			statusResponse, err := imagesClient.Status(ctx, &statusRequest)
			logrus.Debugf("%#v", statusResponse.Image)
			if err == nil && statusResponse.Image != nil {
				req.Image = statusRequest.Image
			}
		}
		res, err := imagesClient.Remove(ctx, req)
		if err != nil {
			return err
		}
		logrus.Debugf("%#v", res)
		return nil
	})
}
