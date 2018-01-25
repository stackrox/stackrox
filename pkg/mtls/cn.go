package mtls

import (
	"fmt"
	"strings"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
)

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
