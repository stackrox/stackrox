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

type (
	loadCertificateFunc   func(dir string) (*tls.Certificate, error)
	updateCertificateFunc func(cert *tls.Certificate)
)

// WatchCertDir starts watching the directory containing certificates.
// The name parameter is used in log messages to identify which certificate is being watched.
func WatchCertDir(name string, dir string, loadCert loadCertificateFunc, updateCert updateCertificateFunc, options ...CertWatchOption) {
	wh := &handler{
		name:       name,
		dir:        dir,
		loadCert:   loadCert,
		updateCert: updateCert,
		cfg:        applyOptions(options...),
	}

	watchOpts := k8scfgwatch.Options{
		Interval: watchInterval,
		Force:    true,
	}
	_ = k8scfgwatch.WatchConfigMountDir(context.Background(), dir, k8scfgwatch.DeduplicateWatchErrors(wh), watchOpts)
}

type handler struct {
	name       string
	dir        string
	loadCert   loadCertificateFunc
	updateCert updateCertificateFunc
	cfg        *CertWatchConfig
}

func (h *handler) OnChange(dir string) (interface{}, error) {
	return h.loadCert(dir)
}

func (h *handler) OnStableUpdate(val interface{}, err error) {
	var cert *tls.Certificate
	if err != nil {
		log.Errorf("Error reading %s certificate: %v. Watch dir: %q", h.name, err, h.dir)
	} else {
		cert, _ = val.(*tls.Certificate)
		if cert == nil {
			log.Infof("No %s certificate found in %q", h.name, h.dir)
		} else {
			log.Infof("%s certificate loaded (SerialNumber: %s, Subject: %s, DNSNames: %s, Issuer: %s), watch dir: %q",
				h.name, cert.Leaf.SerialNumber, cert.Leaf.Subject, cert.Leaf.DNSNames, cert.Leaf.Issuer, h.dir)

			parsedChain, err := x509utils.ParseCertificateChain(cert.Certificate)
			if err != nil {
				log.Errorf("Error parsing certificate #%d in the certificate chain (dir: %q): %v", len(parsedChain), h.dir, err)
			} else if h.cfg.GetVerify() {
				if err := x509utils.VerifyCertificateChain(parsedChain, x509.VerifyOptions{}); err != nil {
					log.Warnf("This server does not trust its own certificate! "+
						"If you see certificate trust issues in your clients (e.g. sensors), "+
						"please ensure that your certificate PEM file includes the entire certificate chain. "+
						"Watch dir: %q", h.dir)
				}
			}
		}
	}

	h.updateCert(cert)
}

func (h *handler) OnWatchError(err error) {
	log.Errorf("Error watching %s certificate directory %q: %v", h.name, h.dir, err)
}
