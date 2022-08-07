package servicecerttoken

import (
	"crypto/x509"
	"strings"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/types"
	ctTLS "github.com/google/certificate-transparency-go/tls"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/grpc/client/authn/servicecerttoken"
)

// parseToken parses a ServiceCert token and returns the parsed x509 certificate. Note that the returned certificate is
// not verified.
func parseToken(token string, maxLeeway time.Duration) (*x509.Certificate, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return nil, errors.Errorf("expected token to contain exactly one '.', found %d", len(parts)-1)
	}

	authBytes, err := servicecerttoken.TokenB64Enc.DecodeString(parts[0])
	if err != nil {
		return nil, errors.Wrap(err, "could not decode auth info")
	}
	sigBytes, err := servicecerttoken.TokenB64Enc.DecodeString(parts[1])
	if err != nil {
		return nil, errors.Wrap(err, "could not decode signature")
	}

	var auth central.ServiceCertAuth
	if err := proto.Unmarshal(authBytes, &auth); err != nil {
		return nil, errors.Wrap(err, "could not unmarshal service cert auth structure")
	}

	ts, err := types.TimestampFromProto(auth.GetCurrentTime())
	if err != nil {
		return nil, errors.Wrap(err, "could not convert timestamp")
	}
	tsDiff := time.Since(ts)
	if tsDiff < 0 {
		tsDiff = -tsDiff
	}
	if tsDiff > maxLeeway {
		return nil, errors.Errorf("time difference %v > %v detected", tsDiff, maxLeeway)
	}

	cert, err := x509.ParseCertificate(auth.GetCertDer())
	if err != nil {
		return nil, errors.Wrap(err, "could not parse certificate data")
	}

	ds := ctTLS.DigitallySigned{
		Algorithm: ctTLS.SignatureAndHashAlgorithm{
			Hash:      servicecerttoken.HashAlgo,
			Signature: ctTLS.SignatureAlgorithmFromPubKey(cert.PublicKey),
		},
		Signature: sigBytes,
	}

	if err := ctTLS.VerifySignature(cert.PublicKey, authBytes, ds); err != nil {
		return nil, errors.Wrap(err, "failed to verify signature")
	}
	return cert, nil
}
