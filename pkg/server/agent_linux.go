package server

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net"
	"time"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/api/events"
	"github.com/containerd/containerd/content"
	"github.com/containerd/containerd/errdefs"
	"github.com/containerd/containerd/images"
	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/containerd/platforms"
	"github.com/containerd/typeurl"
	"github.com/gogo/protobuf/types"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
	imagesv1 "github.com/rancher/kim/pkg/apis/services/images/v1alpha1"
	"github.com/rancher/kim/pkg/client"
	imgsvr "github.com/rancher/kim/pkg/server/images"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func (a *Agent) Run(ctx context.Context) error {
	backend, err := a.Interface(ctx, &client.DefaultConfig)
	if err != nil {
		return err
	}
	defer backend.Close()

	go a.syncImageContent(namespaces.WithNamespace(ctx, buildkitNamespace), backend.Containerd)
	go a.listenAndServe(ctx, backend)

	select {
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (a *Agent) listenAndServe(ctx context.Context, backend *imgsvr.Server) error {
	lc := &net.ListenConfig{}
	listener, err := lc.Listen(ctx, "tcp", fmt.Sprintf("0.0.0.0:%d", a.AgentPort))
	if err != nil {
		return err
	}
	defer listener.Close()

	var serverOptions []grpc.ServerOption
	if a.Tlscert != "" && a.Tlskey != "" && a.Tlscacert != "" {
		serverCert, err := tls.LoadX509KeyPair(a.Tlscert, a.Tlskey)
		if err != nil {
			return err
		}
		tlsConfig := &tls.Config{
			Certificates: []tls.Certificate{serverCert},
			ClientAuth:   tls.NoClientCert,
		}
		if a.Tlscacert != "" {
			caCert, err := ioutil.ReadFile(a.Tlscacert)
			if err != nil {
				return err
			}
			tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
			tlsConfig.ClientCAs = x509.NewCertPool()
			if ok := tlsConfig.ClientCAs.AppendCertsFromPEM(caCert); !ok {
				return errors.New("failed to append ca certificate")
			}
		}
		serverOptions = append(serverOptions, grpc.Creds(credentials.NewTLS(tlsConfig)))
	}
	server := grpc.NewServer(serverOptions...)
	imagesv1.RegisterImagesServer(server, backend)
	defer server.Stop()
	return server.Serve(listener)
}

func (a *Agent) syncImageContent(ctx context.Context, ctr *containerd.Client) {
	events, errors := ctr.EventService().Subscribe(ctx, `topic~="/images/"`)
	for {
		select {
		case <-ctx.Done():
			return
		case err, ok := <-errors:
			if !ok {
				return
			}
			logrus.Errorf("sync-image-content: %v", err)
		case evt, ok := <-events:
			if !ok {
				return
			}
			if evt.Namespace != buildkitNamespace {
				continue
			}
			if err := handleImageEvent(ctx, ctr, evt.Event); err != nil {
				logrus.Errorf("sync-image-content: handling %#v returned %v", evt, err)
			}
		}
	}
}

func handleImageEvent(ctx context.Context, ctr *containerd.Client, any *types.Any) error {
	evt, err := typeurl.UnmarshalAny(any)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal any")
	}

	switch e := evt.(type) {
	case *events.ImageCreate:
		logrus.Debugf("image-create: %s", e.Name)
		return copyImageContent(ctx, ctr, e.Name, func(x context.Context, s images.Store, i images.Image) error {
			_, err := s.Create(x, i)
			if errdefs.IsAlreadyExists(err) {
				_, err = s.Update(x, i)
			}
			return err
		})
	case *events.ImageUpdate:
		logrus.Debugf("image-update: %s", e.Name)
		return copyImageContent(ctx, ctr, e.Name, func(x context.Context, s images.Store, i images.Image) error {
			_, err := s.Create(x, i)
			if errdefs.IsAlreadyExists(err) {
				_, err = s.Update(x, i)
			}
			return err
		})
	}

	return nil
}

func copyImageContent(ctx context.Context, ctr *containerd.Client, name string, fn func(context.Context, images.Store, images.Image) error) error {
	imageStore := ctr.ImageService()
	img, err := imageStore.Get(ctx, name)
	if err != nil {
		return err
	}
	contentStore := ctr.ContentStore()
	fromCtx, fromCancel := context.WithTimeout(ctx, time.Minute*3)
	defer fromCancel()
	toCtx := namespaces.WithNamespace(fromCtx, "k8s.io")
	handler := images.Handlers(images.ChildrenHandler(contentStore), copyImageContentHandler(toCtx, contentStore, img))
	if err = images.Walk(fromCtx, handler, img.Target); err != nil {
		return err
	}
	return fn(toCtx, imageStore, img)
}

func copyImageContentHandler(toCtx context.Context, contentStore content.Store, img images.Image) images.HandlerFunc {
	return func(fromCtx context.Context, desc ocispec.Descriptor) (children []ocispec.Descriptor, err error) {
		logrus.Debugf("copy-image-content: media-type=%v, digest=%v", desc.MediaType, desc.Digest)
		info, err := waitImageContentInfo(fromCtx, contentStore, desc)
		if err != nil {
			return children, err
		}
		logrus.Debugf("copy-image-content: info=%#v", info)
		ra, err := contentStore.ReaderAt(fromCtx, desc)
		if err != nil {
			return children, err
		}
		defer ra.Close()
		wopts := []content.WriterOpt{content.WithRef(img.Name)}
		if _, err := contentStore.Info(toCtx, desc.Digest); errdefs.IsNotFound(err) {
			// if the image does not already exist in the target namespace we supply the descriptor here to ensure
			// that it is created with proper size information. if the image already exists the size for the digest
			// for the to-be updated image is sourced from what is passed to content.Copy
			wopts = append(wopts, content.WithDescriptor(desc))
		}
		w, err := contentStore.Writer(toCtx, wopts...)
		if err != nil {
			return children, err
		}
		defer w.Close()
		err = content.Copy(toCtx, w, content.NewReader(ra), desc.Size, desc.Digest, content.WithLabels(info.Labels))
		if err != nil && errdefs.IsAlreadyExists(err) {
			return children, nil
		}
		return children, err
	}
}

// waitImageContentInfo waits for all referenced content to become available because buildkit can trigger image
// creation events before all referenced content is fully available (via unpack)
func waitImageContentInfo(ctx context.Context, contentStore content.Store, desc ocispec.Descriptor) (content.Info, error) {
	available, _, _, missing, err := images.Check(ctx, contentStore, desc, platforms.Default())
	if err != nil {
		return content.Info{}, err
	}
	if !available && len(missing) > 0 {
		for _, m := range missing {
			logrus.Debugf("wait-image-content: missing=%#v", m)
		next:
			for {
				select {
				case <-ctx.Done():
					return content.Info{}, ctx.Err()
				case <-time.After(100 * time.Millisecond):
					info, err := contentStore.Info(ctx, desc.Digest)
					if err == nil {
						logrus.Debugf("wait-image-content: available=%#v", info)
						break next
					}
					if !errdefs.IsNotFound(err) {
						return content.Info{}, err
					}
				}
			}
		}
	}
	return contentStore.Info(ctx, desc.Digest)
}
