package generate

import (
	"crypto/tls"

	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/renderer"
)

var (
	cfg renderer.Config
)

func validateConfig(c *renderer.Config) error {
	if err := validateHostPath(c.HostPath); err != nil {
		return err
	}
	if err := validateDefaultTLSCert(c.DefaultTLSCertPEM, c.DefaultTLSKeyPEM); err != nil {
		return err
	}
	return nil
}

func validateHostPath(hostpath *renderer.HostPathPersistence) error {
	if hostpath == nil {
		return nil
	}
	if err := validateHostPathInstance(hostpath.Central); err != nil {
		return err
	}
	return validateHostPathInstance(hostpath.DB)
}

func validateHostPathInstance(instance *renderer.HostPathPersistenceInstance) error {
	if instance == nil {
		return nil
	}
	if (instance.NodeSelectorKey == "") != (instance.NodeSelectorValue == "") {
		return errox.InvalidArgs.New("Both node selector key and node selector value must be specified when using a hostpath")
	}
	return nil
}

func validateDefaultTLSCert(certPEM, keyPEM []byte) error {
	if len(certPEM) == 0 && len(keyPEM) == 0 {
		return nil
	}

	_, err := tls.X509KeyPair(certPEM, keyPEM)
	return err
}
