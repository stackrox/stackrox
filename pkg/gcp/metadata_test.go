package gcp

import (
	"testing"

	"cloud.google.com/go/compute/metadata"
	"github.com/stretchr/testify/assert"
)

func TestNotOnGCP(t *testing.T) {
	if !metadata.OnGCE() {
		_, err := GetMetadata()
		assert.NoError(t, err)
	}
}
