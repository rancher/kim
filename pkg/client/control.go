package client

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	buildkit "github.com/moby/buildkit/client"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ControlFunc func(context.Context, *buildkit.Client) error

func Control(ctx context.Context, k8s *Interface, fn ControlFunc) error {
	addr, err := GetServiceAddress(ctx, k8s, "buildkit")
	if err != nil {
		return err
	}

	tmp, err := ioutil.TempDir("", "kim-tls-*")
	if err != nil {
		return errors.Wrap(err, "failed to create temp directory")
	}
	defer os.RemoveAll(tmp)

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
