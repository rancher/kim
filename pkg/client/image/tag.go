package image

import (
	"context"

	"github.com/docker/distribution/reference"
	imagesv1 "github.com/rancher/kim/pkg/apis/services/images/v1alpha1"
	"github.com/rancher/kim/pkg/client"
	"github.com/sirupsen/logrus"
	criv1 "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
)

type Tag struct {
}

func (s *Tag) Do(ctx context.Context, k8s *client.Interface, image string, tags []string) error {
	if named, err := reference.ParseNormalizedNamed(image); err == nil {
		image = reference.TagNameOnly(named).String()
	}
	normalizedTags := make([]string, len(tags))
	for i, tag := range tags {
		named, err := reference.ParseNormalizedNamed(tag)
		if err != nil {
			return err
		}
		normalizedTags[i] = named.String()
	}
	return client.Images(ctx, k8s, func(ctx context.Context, imagesClient imagesv1.ImagesClient) error {
		req := &imagesv1.ImageTagRequest{
			Image: &criv1.ImageSpec{
				Image: image,
			},
			Tags: normalizedTags,
		}
		res, err := imagesClient.Tag(ctx, req)
		if err != nil {
			return err
		}
		logrus.Debugf("image-tag: %#v", res)
		return nil
	})
}
