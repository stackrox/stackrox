//go:build test_all

package mitre

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetBundle(t *testing.T) {
	bundle, err := GetMitreBundle()
	assert.NoError(t, err)
	assert.True(t, len(bundle.GetMatrices()) > 0)
}
