package aws

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetMetadata_NotOnAWS(t *testing.T) {
	t.Parallel()

	_, err := GetMetadata()
	// We might not get metadata info, but we should not get an error.
	assert.NoError(t, err)
}
