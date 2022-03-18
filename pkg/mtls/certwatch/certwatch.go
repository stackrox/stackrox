package certwatch

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"strings"
	"time"

	"github.com/stackrox/rox/pkg/k8scfgwatch"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/x509utils"
)

const (
	watchInterval = 5 * time.Second
)

var (
	log = logging.LoggerForModule()
)

type loadCertificateFunc func(dir string) (*tls.Certificate, error)
type updateCertificateFunc func(cert *tls.Certificate)

// WatchCertDir starts watching the directory containing certificates
func WatchCertDir(certChain string, dir string, loadCert loadCertificateFunc, updateCert updateCertificateFunc) {
	wh := &handler{
		certChain:  certChain,
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
	certChain  string
	loadCert   loadCertificateFunc
	updateCert updateCertificateFunc
}

func (h *handler) OnChange(dir string) (interface{}, error) {
	return h.loadCert(dir)
}

func (h *handler) OnStableUpdate(val interface{}, err error) {
	var cert *tls.Certificate
	if err != nil {
		log.Errorf("Error reading %s TLS certificates: %v. Disabling %s TLS certificate.", h.certChain, err, h.certChain)
	} else {
		cert, _ = val.(*tls.Certificate)
		if cert == nil {
			log.Infof("No %s TLS certificate found. Using internal certificates for HTTPS", h.certChain)
		} else {
			log.Infof("%s TLS certificate loaded, using cert with DN %s for HTTPS", strings.Title(h.certChain), cert.Leaf.Subject)

			parsedChain, err := x509utils.ParseCertificateChain(cert.Certificate)
			if err != nil {
				log.Errorf("Error parsing certificate #%d in the %s certificate chain: %v", len(parsedChain), h.certChain, err)
			} else if err := x509utils.VerifyCertificateChain(parsedChain, x509.VerifyOptions{}); err != nil {
				log.Warnf("This server does not trust its own %s certificate! "+
					"If you see certificate trust issues in your clients (e.g. sensors), "+
					"please ensure that your certificate PEM file includes the entire certificate chain.", h.certChain)
			}
		}
	}

	h.updateCert(cert)
}

func (h *handler) OnWatchError(err error) {
	log.Errorf("Error watching %s TLS certificate directory: %v. Not updating TLS certificates!", h.certChain, err)
}
