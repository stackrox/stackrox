package metrics

import (
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"os"

	"github.com/pkg/errors"
)

var (
	fakeClientCAFile   = "./testdata/ca.pem"
	fakeClientCertFile = "./testdata/client.crt"
	fakeClientKeyFile  = "./testdata/client.key"
	fakeCertFile       = "./testdata/tls.crt"
	fakeKeyFile        = "./testdata/tls.key"
)

func testClient() (*http.Client, error) {
	cert, err := tls.LoadX509KeyPair(fakeClientCertFile, fakeClientKeyFile)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load client certificate")
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			Certificates: []tls.Certificate{cert},
			// We are using a self-signed certificate for testing.
			InsecureSkipVerify: true,
		},
	}
	client := &http.Client{Transport: tr}
	return client, nil
}

// FakeTLSConfigurer is a fake TLS configurer with pre-defined certificates.
type FakeTLSConfigurer struct {
	tlsConfig *tls.Config
}

// WatchForChanges watches for changes of the server TLS certificate files and the client CA config map.
func (t *FakeTLSConfigurer) WatchForChanges() {}

// TLSConfig returns the current TLS config.
func (t *FakeTLSConfigurer) TLSConfig() (*tls.Config, error) {
	return t.tlsConfig, nil
}

func newFakeTLSConfigurer() (TLSConfigurer, error) {
	cert, err := tls.LoadX509KeyPair(fakeCertFile, fakeKeyFile)
	if err != nil {
		return nil, errors.Wrap(err, "loading test certificate failed")
	}

	certPool := x509.NewCertPool()
	pem, err := os.ReadFile(fakeClientCAFile)
	if err != nil {
		return nil, errors.Wrap(err, "loading test client CA certificate")
	}
	if !certPool.AppendCertsFromPEM(pem) {
		return nil, errors.Wrap(err, "failed to add client certificate")
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    certPool,
	}
	cfgr := &FakeTLSConfigurer{tlsConfig: tlsConfig}
	return cfgr, nil
}
