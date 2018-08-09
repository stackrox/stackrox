package mtls

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/stackrox/rox/generated/api/v1"
)

// Identity identifies a particular certificate.
type Identity struct {
	Name   CommonName
	Serial *big.Int
}

// V1 returns the identity represented as a v1 API ServiceIdentity.
func (id Identity) V1() *v1.ServiceIdentity {
	return &v1.ServiceIdentity{
		Serial: id.Serial.Int64(),
		Type:   id.Name.ServiceType,
		Id:     id.Name.Identifier,
	}
}

// CommonName encodes the parts of a certificate common name (CN).
type CommonName struct {
	ServiceType v1.ServiceType
	Identifier  string
}

func (c CommonName) String() string {
	return fmt.Sprintf("%s: %s", c.ServiceType, c.Identifier)
}

// CommonNameFromString parses a CN string into its component parts.
func CommonNameFromString(s string) CommonName {
	parts := strings.SplitN(s, ":", 2)
	if len(parts) == 2 {
		return CommonName{
			ServiceType: v1.ServiceType(v1.ServiceType_value[parts[0]]),
			Identifier:  strings.TrimSpace(parts[1]),
		}
	}
	return CommonName{
		Identifier: s,
	}
}
