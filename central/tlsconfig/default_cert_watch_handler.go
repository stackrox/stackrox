package tlsconfig

import (
	"crypto/tls"
	"crypto/x509"

	"github.com/stackrox/rox/pkg/x509utils"
)

type defaultCertWatchHandler struct {
	mgr *managerImpl
}

func (h *defaultCertWatchHandler) OnChange(dir string) (interface{}, error) {
	return loadDefaultCertificate(dir)
}

func (h *defaultCertWatchHandler) OnStableUpdate(val interface{}, err error) {
	var defaultCert *tls.Certificate
	if err != nil {
		log.Errorf("Error reading default TLS certificates: %v. Disabling default TLS certificate.", err)
	} else {
		defaultCert, _ = val.(*tls.Certificate)
		if defaultCert == nil {
			log.Info("No default TLS certificate found. Using internal certificates for HTTPS")
		} else {
			log.Infof("Default TLS certificate loaded, using cert with DN %s for HTTPS", defaultCert.Leaf.Subject)

			parsedChain, err := x509utils.ParseCertificateChain(defaultCert.Certificate)
			if err != nil {
				log.Errorf("Error parsing certificate #%d in the default certificate chain: %v", len(parsedChain), err)
			} else if err := x509utils.VerifyCertificateChain(parsedChain, x509.VerifyOptions{}); err != nil {
				log.Warn("Central does not trust its own default certificate! " +
					"If you see certificate trust issues in your clients (including sensors), please ensure that your certificate PEM file includes the entire certificate chain.")
			}
		}
	}

	h.mgr.UpdateDefaultCert(defaultCert)
}

func (h *defaultCertWatchHandler) OnWatchError(err error) {
	log.Errorf("Error watching default TLS certificate directory: %v. Not updating TLS certificates!", err)
}
