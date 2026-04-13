package docker

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestV2ManifestList_JSONCompat verifies our inline v2ManifestList parses
// the same JSON wire format that docker/distribution's DeserializedManifestList would.
func TestV2ManifestList_JSONCompat(t *testing.T) {
	// Real manifest list JSON from Docker Hub (abbreviated).
	raw := `{
		"schemaVersion": 2,
		"mediaType": "application/vnd.docker.distribution.manifest.list.v2+json",
		"manifests": [
			{
				"mediaType": "application/vnd.docker.distribution.manifest.v2+json",
				"size": 1357,
				"digest": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				"platform": {
					"architecture": "amd64",
					"os": "linux"
				}
			},
			{
				"mediaType": "application/vnd.docker.distribution.manifest.v2+json",
				"size": 1357,
				"digest": "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
				"platform": {
					"architecture": "arm64",
					"os": "linux",
					"variant": "v8"
				}
			}
		]
	}`

	var ml v2ManifestList
	require.NoError(t, json.Unmarshal([]byte(raw), &ml))
	require.Len(t, ml.Manifests, 2)

	assert.Equal(t, "amd64", ml.Manifests[0].Platform.Architecture)
	assert.Equal(t, "linux", ml.Manifests[0].Platform.OS)
	assert.Equal(t, MediaTypeV2Manifest, ml.Manifests[0].MediaType)
	assert.Equal(t, "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", ml.Manifests[0].Digest.String())
	assert.Equal(t, int64(1357), ml.Manifests[0].Size)

	assert.Equal(t, "arm64", ml.Manifests[1].Platform.Architecture)
	assert.Equal(t, "v8", ml.Manifests[1].Platform.Variant)
}

// TestV2Manifest_JSONCompat verifies our inline v2Manifest parses schema2 JSON.
func TestV2Manifest_JSONCompat(t *testing.T) {
	raw := `{
		"schemaVersion": 2,
		"mediaType": "application/vnd.docker.distribution.manifest.v2+json",
		"config": {
			"mediaType": "application/vnd.docker.container.image.v1+json",
			"size": 7023,
			"digest": "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
		},
		"layers": [
			{
				"mediaType": "application/vnd.docker.image.rootfs.diff.tar.gzip",
				"size": 32654,
				"digest": "sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"
			},
			{
				"mediaType": "application/vnd.docker.image.rootfs.diff.tar.gzip",
				"size": 16724,
				"digest": "sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"
			}
		]
	}`

	var m v2Manifest
	require.NoError(t, json.Unmarshal([]byte(raw), &m))
	assert.Equal(t, "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", m.Config.Digest.String())
	require.Len(t, m.Layers, 2)
	assert.Equal(t, "sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd", m.Layers[0].Digest.String())
	assert.Equal(t, int64(32654), m.Layers[0].Size)
}

// TestV1SignedManifest_JSONCompat verifies our inline v1SignedManifest parses schema1 JSON.
func TestV1SignedManifest_JSONCompat(t *testing.T) {
	raw := `{
		"schemaVersion": 1,
		"name": "library/nginx",
		"tag": "latest",
		"architecture": "amd64",
		"fsLayers": [
			{"blobSum": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
			{"blobSum": "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}
		],
		"history": [
			{"v1Compatibility": "{\"id\":\"layer1\",\"created\":\"2024-01-01T00:00:00Z\"}"},
			{"v1Compatibility": "{\"id\":\"layer2\",\"created\":\"2024-01-02T00:00:00Z\"}"}
		]
	}`

	var m v1SignedManifest
	require.NoError(t, json.Unmarshal([]byte(raw), &m))
	require.Len(t, m.FSLayers, 2)
	assert.Equal(t, "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", m.FSLayers[0].BlobSum.String())
	require.Len(t, m.History, 2)
	assert.Contains(t, m.History[0].V1Compatibility, "layer1")
}

// TestMediaTypeConstants verifies our inlined constants match the docker/distribution originals.
func TestMediaTypeConstants(t *testing.T) {
	// These are the canonical values from the Docker Registry V2 spec.
	// They must never change or registries won't recognize our Accept headers.
	assert.Equal(t, "application/vnd.docker.distribution.manifest.v1+json", MediaTypeV1Manifest)
	assert.Equal(t, "application/vnd.docker.distribution.manifest.v1+prettyjws", MediaTypeV1SignedManifest)
	assert.Equal(t, "application/vnd.docker.distribution.manifest.list.v2+json", MediaTypeV2ManifestList)
	assert.Equal(t, "application/vnd.docker.distribution.manifest.v2+json", MediaTypeV2Manifest)
	assert.Equal(t, "application/vnd.oci.image.index.v1+json", MediaTypeImageIndex)
	assert.Equal(t, "application/vnd.oci.image.manifest.v1+json", MediaTypeImageManifest)
}

// TestPlatformSpec_OSVersion verifies the os.version JSON tag (note the dot, not underscore).
func TestPlatformSpec_OSVersion(t *testing.T) {
	raw := `{
		"manifests": [{
			"mediaType": "application/vnd.docker.distribution.manifest.v2+json",
			"size": 100,
			"digest": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			"platform": {
				"architecture": "amd64",
				"os": "windows",
				"os.version": "10.0.17763.5458",
				"os.features": ["win32k"]
			}
		}]
	}`

	var ml v2ManifestList
	require.NoError(t, json.Unmarshal([]byte(raw), &ml))
	require.Len(t, ml.Manifests, 1)
	assert.Equal(t, "10.0.17763.5458", ml.Manifests[0].Platform.OSVersion)
	assert.Equal(t, []string{"win32k"}, ml.Manifests[0].Platform.OSFeatures)
}
