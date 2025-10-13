package generate

import (
	"crypto/tls"

	"github.com/pkg/errors"
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
	return validateHostPathInstance(hostpath.DB)
}

func validateHostPathInstance(instance *renderer.HostPathPersistenceInstance) error {
	if instance == nil {
		return nil
	}
	if instance.HostPath == "" {
		return errox.InvalidArgs.New("Non-empty HostPath must be specified")
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
	return errors.Wrap(err, "validating TLS certificate and key pair")
}
