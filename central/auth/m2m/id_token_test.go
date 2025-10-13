package m2m

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIssuerFromRawIDToken(t *testing.T) {
	// Example token taken from: https://pkg.go.dev/github.com/golang-jwt/jwt/v5#example-ParseWithClaims-CustomClaimsType.
	//#nosec G101 -- This is a static example JWT token for testing purposes.
	rawIDToken := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJmb28iOiJiYXIiLCJpc3MiOiJ0ZXN0IiwiYXVkIjoic2luZ2xlIn0.QAWg1vGvnqRuCFTMcPkjZljXHh8U3L_qUjszOtQbeaA"

	issuer, err := IssuerFromRawIDToken(rawIDToken)
	assert.NoError(t, err)
	assert.Equal(t, "test", issuer)
}
