// Package certwatch provides functions to monitor certificate directory updates at runtime.
package certwatch

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"time"

	"github.com/stackrox/rox/pkg/k8scfgwatch"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/x509utils"
)

const (
	watchInterval = 5 * time.Second
)

var (
	log                     = logging.LoggerForModule()
	_   k8scfgwatch.Handler = (*handler)(nil) // compile time check that the handler implements the interface
)

type loadCertificateFunc func(dir string) (*tls.Certificate, error)
type updateCertificateFunc func(cert *tls.Certificate)

// WatchCertDir starts watching the directory containing certificates
func WatchCertDir(dir string, loadCert loadCertificateFunc, updateCert updateCertificateFunc) {
	wh := &handler{
		dir:        dir,
		loadCert:   loadCert,
		updateCert: updateCert,
	}

	watchOpts := k8scfgwatch.Options{
		Interval: watchInterval,
		Force:    true,
	}
	_ = k8scfgwatch.WatchConfigMountDir(context.Background(), dir, k8scfgwatch.DeduplicateWatchErrors(wh), watchOpts)
}

type handler struct {
	dir        string
	loadCert   loadCertificateFunc
	updateCert updateCertificateFunc
}

func (h *handler) OnChange(dir string) (interface{}, error) {
	return h.loadCert(dir)
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
			log.Infof("TLS certificate loaded, using the following cert for HTTPS: (SerialNumber: %s, Subject: %s, DNSNames, %s), watch dir: %q",
				cert.Leaf.SerialNumber, cert.Leaf.Subject, cert.Leaf.DNSNames, h.dir)

			parsedChain, err := x509utils.ParseCertificateChain(cert.Certificate)
			if err != nil {
				log.Errorf("Error parsing certificate #%d in the certificate chain (dir: %q): %v", len(parsedChain), h.dir, err)
			} else if err := x509utils.VerifyCertificateChain(parsedChain, x509.VerifyOptions{}); err != nil {
				log.Warnf("This server does not trust its own certificate! "+
					"If you see certificate trust issues in your clients (e.g. sensors), "+
					"please ensure that your certificate PEM file includes the entire certificate chain."+
					"Watch dir: %q", h.dir)
			}
		}
	}

	h.updateCert(cert)
}

func (h *handler) OnWatchError(err error) {
	log.Errorf("Error watching TLS certificate directory %q: %v. Not updating TLS certificates", h.dir, err)
}
