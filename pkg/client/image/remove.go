package image

import (
	"context"

	"github.com/pkg/errors"
	imagesv1 "github.com/rancher/kim/pkg/apis/services/images/v1alpha1"
	"github.com/rancher/kim/pkg/client"
	"github.com/sirupsen/logrus"
)

type Remove struct {
}

func (s *Remove) Do(ctx context.Context, k8s *client.Interface, image string) error {
	return client.Images(ctx, k8s, func(ctx context.Context, imagesClient imagesv1.ImagesClient) error {
		ref, err := refSpec(ctx, imagesClient, image)
		if err != nil {
			return err
		}
		if ref == nil {
			return errors.Errorf("image %q: not found", image)
		}
		res, err := imagesClient.Remove(ctx, &imagesv1.ImageRemoveRequest{Image: ref})
		logrus.Debugf("%#v", res)
		return err
	})
}
