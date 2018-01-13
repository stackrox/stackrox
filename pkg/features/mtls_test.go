package features

import (
	"testing"
)

func TestMTLS(t *testing.T) {
	for _, test := range defaultTrueCases {
		testFlagEnabled(t, MTLS, MTLS, test)
	}
}
