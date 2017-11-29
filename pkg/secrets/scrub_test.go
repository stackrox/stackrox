package secrets

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestScrubSecrets(t *testing.T) {
	m := map[string]string{
		"password": "password",
		"token":    "token",
		"Token":    "token",
		"endpoint": "endpoint",
	}
	assert.Equal(t, map[string]string{"endpoint": "endpoint"}, ScrubSecrets(m))
}
