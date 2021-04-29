package image

import (
	"context"

	"github.com/docker/distribution/reference"
	imagesv1 "github.com/rancher/kim/pkg/apis/services/images/v1alpha1"
	"github.com/rancher/kim/pkg/client"
	"github.com/sirupsen/logrus"
)

type Tag struct {
}

func (s *Tag) Do(ctx context.Context, k8s *client.Interface, image string, tags []string) error {
	normalizedTags := make([]string, len(tags))
	for i, tag := range tags {
		named, err := reference.ParseNormalizedNamed(tag)
		if err != nil {
			return err
		}
		normalizedTags[i] = reference.TagNameOnly(named).String()
	}
	return client.Images(ctx, k8s, func(ctx context.Context, imagesClient imagesv1.ImagesClient) error {
		ref, err := refSpec(ctx, imagesClient, image)
		if err != nil {
			return err
		}
		req := &imagesv1.ImageTagRequest{
			Image: ref,
			Tags:  normalizedTags,
		}
		res, err := imagesClient.Tag(ctx, req)
		if err != nil {
			return err
		}
		logrus.Debugf("image-tag: %#v", res)
		return nil
	})
}
