package utils

import (
	"crypto/tls"
	"crypto/x509"
	"os"

	"github.com/pkg/errors"
)

func NewTLSConfig(caPath string) (*tls.Config, error) {
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	rootCAs := x509.NewCertPool()
	pem, err := os.ReadFile(caPath)
	if err != nil {
		return nil, errors.Wrap(err, "read CA file")
	}

	if !rootCAs.AppendCertsFromPEM(pem) {
		return nil, errors.New("failed to parse custom CA")
	}

	tlsConfig.RootCAs = rootCAs

	return tlsConfig, nil
}
