package image

import (
	"context"
	"io"
	"os"

	"github.com/docker/distribution/reference"
	"github.com/pkg/errors"
	imagesv1 "github.com/rancher/kim/pkg/apis/services/images/v1alpha1"
	"github.com/rancher/kim/pkg/client"
	"github.com/rancher/kim/pkg/progress"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	criv1 "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
)

type Push struct {
}

func (s *Push) Do(ctx context.Context, k8s *client.Interface, image string) error {
	named, err := reference.ParseNormalizedNamed(image)
	if err != nil {
		return errors.Wrap(err, "Failed to parse image")
	}
	image = reference.TagNameOnly(named).String()
	return client.Images(ctx, k8s, func(ctx context.Context, imagesClient imagesv1.ImagesClient) error {
		ch := make(chan []imagesv1.ImageStatus)
		eg, ctx := errgroup.WithContext(ctx)
		// render output from the channel
		eg.Go(func() error {
			return progress.Display(ch, os.Stdout)
		})
		// render progress to the channel
		eg.Go(func() error {
			defer close(ch)
			ppc, err := imagesClient.PushProgress(ctx, &imagesv1.ImageProgressRequest{Image: image})
			if err != nil {
				return err
			}
			for {
				info, err := ppc.Recv()
				if err == io.EOF {
					return nil
				}
				if err != nil {
					return err
				}
				ch <- info.Status
			}
			return nil
		})
		// initiate the push
		eg.Go(func() error {
			req := &imagesv1.ImagePushRequest{
				Image: &criv1.ImageSpec{
					Image: image,
				},
			}
			keyring := client.GetDockerKeyring(ctx, k8s)
			if auth, ok := keyring.Lookup(image); ok {
				req.Auth = &criv1.AuthConfig{
					Username:      auth[0].Username,
					Password:      auth[0].Password,
					Auth:          auth[0].Auth,
					ServerAddress: auth[0].ServerAddress,
					IdentityToken: auth[0].IdentityToken,
					RegistryToken: auth[0].RegistryToken,
				}
			}
			res, err := imagesClient.Push(ctx, req)
			logrus.Debugf("image-push: %v", res)
			return err
		})
		return eg.Wait()
	})
}
