package images

import (
	"context"
	"fmt"
	"strings"

	"github.com/containerd/containerd/images"
	"github.com/containerd/containerd/namespaces"
	imagesv1 "github.com/rancher/kim/pkg/apis/services/images/v1alpha1"
	"github.com/sirupsen/logrus"
)

// Remove image server-side impl
func (s *Server) Remove(ctx context.Context, req *imagesv1.ImageRemoveRequest) (*imagesv1.ImageRemoveResponse, error) {
	logrus.Debugf("image-remove: req=%s", req)
	ctx, done, err := s.Containerd.WithLease(namespaces.WithNamespace(ctx, "k8s.io"))
	if err != nil {
		return nil, err
	}
	defer done(ctx)
	if req.Image == nil {
		return &imagesv1.ImageRemoveResponse{}, nil
	}
	img, err := s.Containerd.ImageService().Get(ctx, req.Image.Image)
	if err != nil {
		return nil, err
	}
	refs := []string{img.Name}
	tags, err := s.Containerd.ImageService().List(ctx, fmt.Sprintf("target.digest==%s,name!=%s", img.Target.Digest, img.Name))
	if err != nil {
		return nil, err
	}
	switch {
	case len(tags) == 1: // single tag
		refs = append(refs, tags[0].Name)
	case strings.HasPrefix(img.Name, "sha256:"): // image id
		for _, tag := range tags {
			refs = append(refs, tag.Name)
		}
	}
	for _, ref := range refs {
		logrus.Debugf("image-remove: ref=%s, img=%#v", ref, req.Image)
		err = s.Containerd.ImageService().Delete(ctx, ref, images.SynchronousDelete())
		if err != nil {
			return nil, err
		}
	}
	return &imagesv1.ImageRemoveResponse{}, nil
}
