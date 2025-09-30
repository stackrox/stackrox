package m2m

import (
	"testing"

	"github.com/stackrox/rox/pkg/errox"
	"github.com/stretchr/testify/assert"
)

func TestIssuerFromRawIDToken(t *testing.T) {
	t.Run("JWT", func(t *testing.T) {
		// Example token taken from: https://pkg.go.dev/github.com/golang-jwt/jwt/v5#example-ParseWithClaims-CustomClaimsType.
		//#nosec G101 -- This is a static example JWT token for testing purposes.
		rawIDToken := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJmb28iOiJiYXIiLCJpc3MiOiJ0ZXN0IiwiYXVkIjoic2luZ2xlIn0.QAWg1vGvnqRuCFTMcPkjZljXHh8U3L_qUjszOtQbeaA"

		issuer, err := IssuerFromRawIDToken(rawIDToken)
		assert.NoError(t, err)
		assert.Equal(t, "test", issuer)
	})

	t.Run("kubernetes opaque token", func(t *testing.T) {
		//#nosec G101 -- This is a static example Kubernetes opaque token for testing purposes.
		rawIDToken := "sha256~98ea6e4f216f2fb4b69fff9b3a44842c38686ca685f3f55dc48c5d3fb1107be4"

		issuer, err := IssuerFromRawIDToken(rawIDToken)
		assert.NoError(t, err)
		assert.Equal(t, KubernetesDefaultSvcTokenIssuer, issuer)
	})

	t.Run("error", func(t *testing.T) {
		//#nosec G101 -- This is an invalid token for testing purposes.
		rawIDToken := "notavalidtoken"

		issuer, err := IssuerFromRawIDToken(rawIDToken)
		assert.ErrorIs(t, err, errox.InvalidArgs)
		assert.Empty(t, issuer)
	})
}
