package cert

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math"
	"math/big"
	"net"
	"strings"
	"time"

	"k8s.io/client-go/util/cert"
	"k8s.io/client-go/util/keyutil"
)

func NewPrivateKey() (crypto.Signer, error) {
	return ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
}

func NewCA(cn string, org ...string) (*x509.Certificate, crypto.Signer, error) {
	key, err := NewPrivateKey()
	if err != nil {
		return nil, nil, err
	}
	if len(org) == 0 {
		org = append(org, cn)
	}
	crt, err := cert.NewSelfSignedCACert(cert.Config{
		CommonName:   cn,
		Organization: org,
	}, key)
	if err != nil {
		return nil, nil, err
	}

	return crt, key, nil
}

type NewCertFunc func(self crypto.Signer, ca *x509.Certificate, signer crypto.Signer, cn string) (*x509.Certificate, error)

var _ NewCertFunc = NewSignedClientCert

func NewSignedClientCert(signee crypto.Signer, issuer *x509.Certificate, signer crypto.Signer, cn string) (*x509.Certificate, error) {
	serial, err := rand.Int(rand.Reader, new(big.Int).SetInt64(math.MaxInt64))
	if err != nil {
		return nil, err
	}

	stub := x509.Certificate{
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		NotAfter:     time.Now().Add(time.Hour * 24 * 365).UTC(),
		NotBefore:    issuer.NotBefore,
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName: cn,
		},
	}

	parts := strings.Split(cn, ",o=")
	if len(parts) > 1 {
		stub.Subject.CommonName = parts[0]
		stub.Subject.Organization = parts[1:]
	}

	cert, err := x509.CreateCertificate(rand.Reader, &stub, issuer, signee.Public(), signer)
	if err != nil {
		return nil, err
	}
	return x509.ParseCertificate(cert)
}

func NewSignedCertFunc(orgs []string, domains []string, ips []net.IP) NewCertFunc {
	return func(signee crypto.Signer, issuer *x509.Certificate, signer crypto.Signer, cn string) (*x509.Certificate, error) {
		serial, err := rand.Int(rand.Reader, new(big.Int).SetInt64(math.MaxInt64))
		if err != nil {
			return nil, err
		}
		stub := x509.Certificate{
			DNSNames:     domains,
			ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
			IPAddresses:  ips,
			KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
			NotAfter:     time.Now().Add(time.Hour * 24 * 365).UTC(),
			NotBefore:    issuer.NotBefore,
			SerialNumber: serial,
			Subject: pkix.Name{
				CommonName:   cn,
				Organization: orgs,
			},
		}
		cert, err := x509.CreateCertificate(rand.Reader, &stub, issuer, signee.Public(), signer)
		if err != nil {
			return nil, err
		}
		return x509.ParseCertificate(cert)
	}
}

func Marshal(cert *x509.Certificate, key crypto.Signer) ([]byte, []byte, error) {
	bytes, err := keyutil.MarshalPrivateKeyToPEM(key)
	if err != nil {
		return nil, nil, err
	}

	return pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert.Raw,
	}), bytes, nil
}

func Unmarshal(certPem, keyPem []byte) (*x509.Certificate, crypto.Signer, error) {
	key, err := keyutil.ParsePrivateKeyPEM(keyPem)
	if err != nil {
		return nil, nil, err
	}

	signer, ok := key.(crypto.Signer)
	if !ok {
		return nil, nil, fmt.Errorf("key is not a crypto.Signer")
	}

	certs, err := cert.ParseCertsPEM(certPem)
	if err != nil {
		return nil, nil, err
	}

	return certs[0], signer, nil
}
