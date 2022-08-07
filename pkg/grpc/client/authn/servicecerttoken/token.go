package servicecerttoken

import (
	"crypto/tls"
	"fmt"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/types"
	ctTLS "github.com/google/certificate-transparency-go/tls"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/cryptoutils"
)

// createToken creates a ServiceCert token from the given certificate, stamping it with the given current timestamp.
func createToken(cert *tls.Certificate, currTime time.Time) (string, error) {
	tsPb, err := types.TimestampProto(currTime)
	if err != nil {
		return "", errors.Wrap(err, "could not create timestamp proto")
	}

	auth := &central.ServiceCertAuth{
		CertDer:     cert.Certificate[0],
		CurrentTime: tsPb,
	}

	authBytes, err := proto.Marshal(auth)
	if err != nil {
		return "", errors.Wrap(err, "could not marshal service cert auth structure")
	}

	ds, err := ctTLS.CreateSignature(cryptoutils.DerefPrivateKey(cert.PrivateKey), HashAlgo, authBytes)
	if err != nil {
		return "", errors.Wrap(err, "could not create signature")
	}

	return fmt.Sprintf("%s.%s", TokenB64Enc.EncodeToString(authBytes), TokenB64Enc.EncodeToString(ds.Signature)), nil
}
