package signatures

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBundleJSONIsValid(t *testing.T) {
	data, err := os.ReadFile("../../image/rhel/redhat-signing-keys/bundle.json")
	require.NoError(t, err, "bundle.json must exist in image/rhel/redhat-signing-keys/")

	bundle, err := ParseKeyBundle(data)
	require.NoError(t, err, "bundle.json must be valid")
	assert.NotEmpty(t, bundle.Keys, "bundle.json must contain at least one key")
}

func TestBundleToSignatureIntegration(t *testing.T) {
	data, err := os.ReadFile("../../image/rhel/redhat-signing-keys/bundle.json")
	require.NoError(t, err)

	bundle, err := ParseKeyBundle(data)
	require.NoError(t, err)

	si := BundleToSignatureIntegration(bundle)
	assert.Equal(t, DefaultRedHatIntegrationID, si.GetId())
	assert.Equal(t, DefaultRedHatIntegrationName, si.GetName())
	assert.Len(t, si.GetCosign().GetPublicKeys(), len(bundle.Keys))
}
