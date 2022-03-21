// Package certwatch provides functions to monitor certificate directory updates at runtime.
package certwatch

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/fileutils"
	"github.com/stackrox/rox/pkg/k8scfgwatch"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/x509utils"
)

const (
	watchInterval = 5 * time.Second
)

var (
	log                     = logging.LoggerForModule()
	_   k8scfgwatch.Handler = (*handler)(nil) // compile time check that the handler implements the interface
)

type updateCertificateFunc func(cert *tls.Certificate)

// WatchMTLSCertDir starts watching the certificate directory
// (or directories if key file and cert file are located in different folders) configured by environment variables.
func WatchMTLSCertDir(configHolder *TLSConfigHolder) error {
	certFilePath, keyFilePath := mtls.CertFilePath(), mtls.KeyFilePath()
	certDir, keyDir := filepath.Dir(certFilePath), filepath.Dir(keyFilePath)

	ca, _, err := mtls.CACert()
	if err != nil {
		return errors.Wrap(err, "CA cert")
	}

	certUpdater := &certUpdater{
		configHolder: configHolder,
		trustRoots:   []*x509.Certificate{ca},
	}

	configHolder.AddServerCertSource(&certUpdater.certs)
	configHolder.AddClientCASource(&certUpdater.trustRoots)

	trustCertPool := x509.NewCertPool()
	for _, rootCA := range certUpdater.trustRoots {
		trustCertPool.AddCert(rootCA)
	}

	if keyDir != certDir {
		WatchCertDir(keyDir, certFilePath, keyFilePath, certUpdater.updateCertificate, trustCertPool)
	}

	WatchCertDir(certDir, certFilePath, keyFilePath, certUpdater.updateCertificate, trustCertPool)

	return nil
}

// WatchCertDir starts watching the directory containing certificates
func WatchCertDir(dir string, certFile string, keyFile string, updateCert updateCertificateFunc, trustRoots *x509.CertPool) {
	wh := &handler{
		dir:        dir,
		certFile:   certFile,
		keyFile:    keyFile,
		updateCert: updateCert,
		trustRoots: trustRoots,
	}

	watchOpts := k8scfgwatch.Options{
		Interval: watchInterval,
		Force:    true,
	}
	_ = k8scfgwatch.WatchConfigMountDir(context.Background(), dir, k8scfgwatch.DeduplicateWatchErrors(wh), watchOpts)
}

type handler struct {
	dir        string
	certFile   string
	keyFile    string
	updateCert updateCertificateFunc
	trustRoots *x509.CertPool
}

func (h *handler) OnChange(_ string) (interface{}, error) {
	return loadCertificate(h.certFile, h.keyFile)
}

func (h *handler) OnStableUpdate(val interface{}, err error) {
	var cert *tls.Certificate
	if err != nil {
		log.Errorf("Error reading TLS certificates: %v. Disabling TLS certificate. Watch dir: %q", err, h.dir)
	} else {
		cert, _ = val.(*tls.Certificate)
		if cert == nil {
			log.Infof("No TLS certificate found. Using internal certificates for HTTPS. Watch dir: %q", h.dir)
		} else {
			log.Infof("TLS certificate loaded, using the following cert for HTTPS: (Subject: %s, DNSNames: %s), watch dir: %q",
				cert.Leaf.Subject, cert.Leaf.DNSNames, h.dir)

			parsedChain, err := x509utils.ParseCertificateChain(cert.Certificate)
			if err != nil {
				log.Errorf("Error parsing certificate #%d in the certificate chain (dir: %q): %v", len(parsedChain), h.dir, err)
			} else if err := x509utils.VerifyCertificateChain(parsedChain, x509.VerifyOptions{Roots: h.trustRoots}); err != nil {
				if h.trustRoots == nil {
					log.Warnf("This server does not trust its own certificate! "+
						"If you see certificate trust issues in your clients (e.g. sensors), "+
						"please ensure that your certificate PEM file includes the entire certificate chain. "+
						"Watch dir: %q, err: %v", h.dir, err)
				} else {
					log.Warnf("This server does not trust its own certificate! "+
						"If you see certificate trust issues in your clients (e.g. sensors), "+
						"please ensure that your certificate PEM file includes one of the trusted roots. "+
						"Watch dir: %q, err: %v", h.dir, err)
				}

			}
		}
	}

	h.updateCert(cert)
}

func (h *handler) OnWatchError(err error) {
	log.Errorf("Error watching TLS certificate directory %q: %v. Not updating TLS certificates!", h.dir, err)
}

func loadCertificate(certFile string, keyFile string) (*tls.Certificate, error) {
	if filesExist, err := fileutils.AllExist(certFile, keyFile); err != nil || !filesExist {
		return nil, err
	}

	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}
	cert.Leaf, err = x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return nil, errors.Wrap(err, "parsing leaf certificate")
	}

	return &cert, nil
}

type certUpdater struct {
	configHolder *TLSConfigHolder
	certs        []tls.Certificate
	trustRoots   []*x509.Certificate
}

func (a *certUpdater) updateCertificate(cert *tls.Certificate) {
	if cert == nil {
		a.certs = nil
	} else {
		a.certs = []tls.Certificate{*cert}
	}

	a.configHolder.UpdateTLSConfig()
}
