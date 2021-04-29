package image

import (
	"context"
	"fmt"
	"strings"

	"github.com/docker/distribution/reference"
	"github.com/sirupsen/logrus"

	imagesv1 "github.com/rancher/kim/pkg/apis/services/images/v1alpha1"
	criv1 "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
)

// refSpec attempts to normalize an arbitrary image reference by requesting the status from the cri with the
// passed value. if it matches or is a image-id prefix then the image-id will be returned. otherwise an attempt is
// made to normalize via reference.ParseNormalizedNamed passed through reference.TagNameOnly (handling tag-less refs)
func refSpec(ctx context.Context, imagesClient imagesv1.ImagesClient, image string) (*criv1.ImageSpec, error) {
	spec := &criv1.ImageSpec{
		Image: image,
	}
	status, statusErr := imagesClient.Status(ctx, &imagesv1.ImageStatusRequest{Image: spec})
	logrus.Debugf("refSpec image=%q: %#v", image, status.Image)
	if statusErr == nil && status.Image != nil && strings.HasPrefix(status.Image.Id, fmt.Sprintf("sha256:%s", image)) {
		spec.Image = status.Image.Id
		return spec, nil
	}
	named, parseErr := reference.ParseNormalizedNamed(image)
	if parseErr == nil {
		spec.Image = reference.TagNameOnly(named).String()
		return spec, nil
	}
	return nil, statusErr
}
