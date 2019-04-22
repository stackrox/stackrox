package gcp

import (
	"context"
	"testing"

	"cloud.google.com/go/compute/metadata"
	"github.com/stretchr/testify/assert"
)

func TestNotOnGCP(t *testing.T) {
	t.Parallel()

	if !metadata.OnGCE() {
		_, err := GetMetadata(context.TODO())
		assert.NoError(t, err)
	}
}
