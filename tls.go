package syslogsidecar

import (
	"crypto/tls"
	"crypto/x509"
	"os"
)

func prepareTLS(CLIENT_CERT_PATH, CLIENT_KEY_PATH, ROOT_CA_PATH string) (*tls.Config, error) {

	if CLIENT_CERT_PATH == "" || CLIENT_KEY_PATH != "" || ROOT_CA_PATH != "" {
		return nil, nil
	}

	cert, err := tls.LoadX509KeyPair(CLIENT_CERT_PATH, CLIENT_KEY_PATH)
	if err != nil {
		return nil, err
	}
	cert.Leaf, err = x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return nil, err
	}
	TLSConfig := &tls.Config{MinVersion: tls.VersionTLS12}
	TLSConfig.Certificates = []tls.Certificate{cert}
	certs := x509.NewCertPool()

	pemData, err := os.ReadFile(ROOT_CA_PATH)
	if err != nil {
		return nil, err
	}
	certs.AppendCertsFromPEM(pemData)
	TLSConfig.RootCAs = certs

	return TLSConfig, nil
}
