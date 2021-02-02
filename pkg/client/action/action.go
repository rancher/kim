package action

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net"
	"path/filepath"
	"strconv"

	buildkit "github.com/moby/buildkit/client"
	"github.com/pkg/errors"
	imagesv1 "github.com/rancher/kim/pkg/apis/services/images/v1alpha1"
	"github.com/rancher/kim/pkg/client"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type DoImagesFunc func(context.Context, imagesv1.ImagesClient) error
type DoControlFunc func(context.Context, *buildkit.Client) error

func DoImages(ctx context.Context, k8s *client.Interface, fn DoImagesFunc) error {
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

func DoControl(ctx context.Context, k8s *client.Interface, fn DoControlFunc) error {
	addr, err := GetServiceAddress(ctx, k8s, "buildkit")
	if err != nil {
		return err
	}

	tmp, err := ioutil.TempDir("", "kim-tls-*")
	if err != nil {
		return errors.Wrap(err, "failed to create temp directory")
	}

	tmpCA := filepath.Join(tmp, "ca.pem")
	tmpCert := filepath.Join(tmp, "cert.pem")
	tmpKey := filepath.Join(tmp, "key.pem")

	options := []buildkit.ClientOpt{
		buildkit.WithCredentials(
			fmt.Sprintf("builder.%s.svc", k8s.Namespace),
			tmpCA, tmpCert, tmpKey,
		),
	}

	// ca
	secret, err := k8s.Core.Secret().Get(k8s.Namespace, "kim-tls-ca", metav1.GetOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to get ca cert")
	}
	if pem, ok := secret.Data[corev1.TLSCertKey]; ok {
		if err = ioutil.WriteFile(tmpCA, pem, 0600); err != nil {
			return errors.Wrap(err, "failed to write temporary ca certificate")
		}
	}
	// client
	secret, err = k8s.Core.Secret().Get(k8s.Namespace, "kim-tls-client", metav1.GetOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to get client cert+key")
	}
	if pem, ok := secret.Data[corev1.TLSCertKey]; ok {
		if err = ioutil.WriteFile(tmpCert, pem, 0600); err != nil {
			return errors.Wrap(err, "failed to write temporary client certificate")
		}
	}
	if pem, ok := secret.Data[corev1.TLSPrivateKeyKey]; ok {
		if err = ioutil.WriteFile(tmpKey, pem, 0600); err != nil {
			return errors.Wrap(err, "failed to write temporary client key")
		}
	}

	bkc, err := buildkit.New(ctx, fmt.Sprintf("tcp://%s", addr), options...)
	if err != nil {
		return err
	}
	defer bkc.Close()
	return fn(ctx, bkc)
}

func GetServiceAddress(_ context.Context, k8s *client.Interface, port string) (string, error) {
	// TODO handle multiple addresses
	endpoints, err := k8s.Core.Endpoints().Get(k8s.Namespace, "builder", metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	for _, sub := range endpoints.Subsets {
		if len(sub.Addresses) > 0 {
			for _, p := range sub.Ports {
				if p.Name == port {
					return net.JoinHostPort(sub.Addresses[0].IP, strconv.FormatInt(int64(p.Port), 10)), nil
				}
			}
		}
	}
	return "", errors.New("unknown service port")
}
