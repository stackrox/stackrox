package secrets

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestScrubSecrets(t *testing.T) {
	m := map[string]string{
		// Don't scrub this:
		"endpoint": "endpoint",

		// Scrub all of these:
		"oauthToken":     "token",
		"oauthtoken":     "token",
		"password":       "password",
		"secretKey":      "secret!",
		"secretkey":      "secret",
		"serviceAccount": "sa",
		"serviceaccount": "sa",
	}
	assert.Equal(t, map[string]string{"endpoint": "endpoint"}, ScrubSecrets(m))
}
