package client

import (
	"crypto"
	"crypto/x509"
	"net"

	"github.com/rancher/kim/pkg/cert"
	corectlv1 "github.com/rancher/wrangler/pkg/generated/controllers/core/v1"
	corev1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

func LoadOrGenCA(secrets corectlv1.SecretClient, namespace, name string) (*x509.Certificate, crypto.Signer, error) {
	secret, err := secrets.Get(namespace, name, metav1.GetOptions{})
	if apierr.IsNotFound(err) {
		secret, err = createAndStoreCA(secrets, namespace, name)
	}
	if err != nil {
		return nil, nil, err
	}
	return cert.Unmarshal(secret.Data[corev1.TLSCertKey], secret.Data[corev1.TLSPrivateKeyKey])
}

func LoadOrGenClientCert(secrets corectlv1.SecretClient, namespace, name string, issuer *x509.Certificate, signer crypto.Signer, cn string) (*x509.Certificate, crypto.Signer, error) {
	secret, err := secrets.Get(namespace, name, metav1.GetOptions{})
	if apierr.IsNotFound(err) {
		secret, err = createAndStoreCert(secrets, namespace, name, cn, issuer, signer, cert.NewSignedClientCert)
	}
	if err != nil {
		return nil, nil, err
	}
	return cert.Unmarshal(secret.Data[corev1.TLSCertKey], secret.Data[corev1.TLSPrivateKeyKey])
}

func LoadOrGenServerCert(secrets corectlv1.SecretClient, namespace, name string, issuer *x509.Certificate, signer crypto.Signer, cn string, orgs, domains []string, ips []net.IP) (*x509.Certificate, crypto.Signer, error) {
	secret, err := secrets.Get(namespace, name, metav1.GetOptions{})
	if apierr.IsNotFound(err) {
		secret, err = createAndStoreCert(secrets, namespace, name, cn, issuer, signer, cert.NewSignedCertFunc(orgs, domains, ips))
	}
	if err != nil {
		return nil, nil, err
	}
	return cert.Unmarshal(secret.Data[corev1.TLSCertKey], secret.Data[corev1.TLSPrivateKeyKey])
}

func createAndStoreCA(secrets corectlv1.SecretClient, namespace, name string) (*corev1.Secret, error) {
	ca, key, err := cert.NewCA(name)
	if err != nil {
		return nil, err
	}
	return marshalAndCreateCert(secrets, namespace, ca, key, name)
}

func createAndStoreCert(secrets corectlv1.SecretClient, namespace, name, cn string, issuer *x509.Certificate, signer crypto.Signer, fn cert.NewCertFunc) (*corev1.Secret, error) {
	key, err := cert.NewPrivateKey()
	if err != nil {
		return nil, err
	}

	crt, err := fn(key, issuer, signer, cn)
	if err != nil {
		return nil, err
	}

	return marshalAndCreateCert(secrets, namespace, crt, key, name)
}

func marshalAndCreateCert(secrets corectlv1.SecretClient, namespace string, crt *x509.Certificate, key crypto.Signer, name string) (*corev1.Secret, error) {
	crtPem, keyPem, err := cert.Marshal(crt, key)
	if err != nil {
		return nil, err
	}

	return secrets.Create(&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: labels.Set{
				"app.kubernetes.io/managed-by": "kim",
			},
		},
		Data: map[string][]byte{
			corev1.TLSCertKey:       crtPem,
			corev1.TLSPrivateKeyKey: keyPem,
		},
		Type: corev1.SecretTypeTLS,
	})
}
