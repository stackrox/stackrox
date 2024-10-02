package storagetov1

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/secrets"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCloudSourceSecretScrub(t *testing.T) {
	storageCloudSource := &storage.CloudSource{
		Credentials: &storage.CloudSource_Credentials{
			Secret: "", ClientId: "id", ClientSecret: "secret",
		},
	}
	v1CloudSource := CloudSource(storageCloudSource)
	require.NotNil(t, v1CloudSource)

	creds := v1CloudSource.GetCredentials()
	assert.Empty(t, creds.GetSecret())
	assert.Equal(t, secrets.ScrubReplacementStr, creds.GetClientId())
	assert.Equal(t, secrets.ScrubReplacementStr, creds.GetClientSecret())
}
