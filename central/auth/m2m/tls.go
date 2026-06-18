package m2m

import (
	"crypto/tls"
	"crypto/x509"
	"os"

	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	serviceCAPathSetting = env.RegisterSetting("ROX_M2M_SERVICE_CA_PATH",
		env.WithDefault("/var/run/secrets/kubernetes.io/serviceaccount/service-ca.crt"))
	k8sCAPathSetting = env.RegisterSetting("ROX_M2M_K8S_CA_PATH",
		env.WithDefault("/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"))
	cnoTrustedCAPathSetting = env.RegisterSetting("ROX_M2M_CNO_CA_PATH",
		env.WithDefault("/etc/pki/injected-ca-trust/tls-ca-bundle.pem"))
)

func tlsConfigWithCustomCertPool() (*tls.Config, error) {
	certPool, err := systemCertPoolWithInjectedCAs()
	if err != nil {
		return nil, err
	}
	return &tls.Config{RootCAs: certPool}, nil
}

func systemCertPoolWithInjectedCAs() (*x509.CertPool, error) {
	certPool, err := x509.SystemCertPool()
	if err != nil {
		return nil, err
	}

	for _, path := range []string{serviceCAPathSetting.Setting(), k8sCAPathSetting.Setting(), cnoTrustedCAPathSetting.Setting()} {
		caBytes, err := os.ReadFile(path)
		if err != nil {
			log.Errorw("Failed to read CA file for token verifier",
				logging.String("path", path), logging.Err(err))
			continue
		}
		if !certPool.AppendCertsFromPEM(caBytes) {
			log.Errorw("Couldn't add CA to cert pool for token verifier",
				logging.String("path", path))
		}
	}

	return certPool, nil
}
