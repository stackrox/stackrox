package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetSource(t *testing.T) {
	// Smoke test the token issuer source creation
	assert.NotNil(t, getTokenSource())
}
