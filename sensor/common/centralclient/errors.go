package centralclient

import (
	"fmt"

	"github.com/pkg/errors"
)

var (
	errMismatchingCentralInstallation = errors.New("using a certificate bundle that was generated from a different Central installation than the one it is trying to connect to")
	errAdditionalCANeeded             = errors.New("the host is using a TLS certificate that is not trusted. Please configure your issuing certificate authority as an additional CA for Sensor")
	errInvalidTrustInfoSignature      = errors.New("invalid trust info signature")
)

func newTrustInfoSignatureErr(reason string) error {
	return fmt.Errorf("%w: %s", errInvalidTrustInfoSignature, reason)
}

func newMismatchCentralErr(reason string) error {
	return fmt.Errorf("%w: %s", errMismatchingCentralInstallation, reason)
}

func newAdditionalCANeededErr(dnsNames []string, hostEndpoint string, message string) error {
	return fmt.Errorf("%w; host (%s) [%v]: %s", errAdditionalCANeeded, hostEndpoint, dnsNames, message)
}
