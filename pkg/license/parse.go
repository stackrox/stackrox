package license

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
)

// EncodeLicenseKey takes the bytes of a serialized license and the corresponding cryptographic signature, and
// returns a printable string representing the license key.
func EncodeLicenseKey(licenseBytes, sigBytes []byte) string {
	return fmt.Sprintf("%s.%s",
		base64.RawStdEncoding.EncodeToString(licenseBytes),
		base64.RawStdEncoding.EncodeToString(sigBytes))
}

// ParseLicenseKey takes a license key (two base64 strings separated by a dot) and returns the raw license data
// (serialized license proto and cryptographic signature).
func ParseLicenseKey(key string) ([]byte, []byte, error) {
	parts := strings.SplitN(key, ".", 3)
	if len(parts) != 2 {
		return nil, nil, errors.New("license key should contain exactly one dot character")
	}

	licenseProtoBytes, err := base64.RawStdEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, nil, errors.Wrap(err, "license is not valid base64 encoded")
	}
	if len(licenseProtoBytes) == 0 {
		return nil, nil, errors.New("license part is missing")
	}

	signatureBytes, err := base64.RawStdEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, nil, errors.Wrap(err, "signature is not valid base64 encoded")
	}
	if len(signatureBytes) == 0 {
		return nil, nil, errors.New("signature part is missing")
	}

	return licenseProtoBytes, signatureBytes, nil
}

// UnmarshalLicense takes a byte slice containing a serialized license proto and unmarshals it, failing if there are any
// extra bytes.
func UnmarshalLicense(licenseBytes []byte) (*v1.License, error) {
	var license v1.License
	if err := proto.Unmarshal(licenseBytes, &license); err != nil {
		return nil, errors.Wrap(err, "could not unmarshal license")
	}

	if restr := license.GetRestrictions(); restr != nil && len(restr.XXX_unrecognized) != 0 {
		return nil, errors.Errorf("could not unmarshal license: %d bytes of unrecognized content", len(restr.XXX_unrecognized))
	}

	return &license, nil
}
