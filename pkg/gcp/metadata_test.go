package gcp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNotOnGCP(t *testing.T) {
	_, err := GetMetadata()
	assert.NoError(t, err)
}
