package do

import (
	"context"
	"crypto/tls"
	"crypto/x509"

	"github.com/pkg/errors"
	imagesv1 "github.com/rancher/kim/pkg/apis/services/images/v1alpha1"
	"github.com/rancher/kim/pkg/client"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ImagesFunc func(context.Context, imagesv1.ImagesClient) error

func Images(ctx context.Context, k8s *client.Interface, fn ImagesFunc) error {
	addr, err := GetServiceAddress(ctx, k8s, "kim")
	if err != nil {
		return err
	}

	tlsConfig := &tls.Config{}

	// ca cert
	secret, err := k8s.Core.Secret().Get(k8s.Namespace, "kim-tls-ca", metav1.GetOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to get ca cert")
	}
	if pem, ok := secret.Data[corev1.TLSCertKey]; ok {
		tlsConfig.RootCAs = x509.NewCertPool()
		tlsConfig.RootCAs.AppendCertsFromPEM(pem)
	}

	// client cert+key
	secret, err = k8s.Core.Secret().Get(k8s.Namespace, "kim-tls-client", metav1.GetOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to get client cert+key")
	}
	certificate, err := tls.X509KeyPair(secret.Data[corev1.TLSCertKey], secret.Data[corev1.TLSPrivateKeyKey])
	if err != nil {
		return errors.Wrap(err, "failed to setup client cert+key")
	}
	tlsConfig.Certificates = []tls.Certificate{certificate}

	conn, err := grpc.DialContext(ctx, addr, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
	if err != nil {
		return err
	}
	defer conn.Close()
	return fn(ctx, imagesv1.NewImagesClient(conn))
}
