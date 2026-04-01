package settingswatch

import (
	"compress/gzip"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/gziputil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecompressAndUnmarshalPolicies(t *testing.T) {
	policies := &storage.PolicyList{
		Policies: []*storage.Policy{
			{
				Id:   "policy-1",
				Name: "Test Policy",
			},
		},
	}

	data, err := policies.MarshalVT()
	require.NoError(t, err)

	compressed, err := gziputil.Compress(data, gzip.BestCompression)
	require.NoError(t, err)

	result, err := decompressAndUnmarshalPolicies(compressed)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Len(t, result.GetPolicies(), 1)
	assert.Equal(t, "policy-1", result.GetPolicies()[0].GetId())
	assert.Equal(t, "Test Policy", result.GetPolicies()[0].GetName())
}

func TestDecompressAndUnmarshalPolicies_InvalidData(t *testing.T) {
	// Note: decompressAndUnmarshalPolicies does not handle empty data gracefully
	// This is expected behavior - the function is only called when data exists
	_, err := decompressAndUnmarshalPolicies([]byte{})
	assert.Error(t, err, "should error on empty data")
}
