package ocprelease

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseImageReferences(t *testing.T) {
	// Minimal shape of release-manifests/image-references (an OpenShift ImageStream).
	refs := []byte(`{
      "kind": "ImageStream",
      "apiVersion": "image.openshift.io/v1",
      "spec": {
        "tags": [
          {"name": "etcd", "from": {"kind": "DockerImage", "name": "quay.io/openshift-release-dev/ocp-v4.0-art-dev@sha256:aaa"}},
          {"name": "cluster-network-operator", "from": {"kind": "DockerImage", "name": "quay.io/openshift-release-dev/ocp-v4.0-art-dev@sha256:bbb"}}
        ]
      }
    }`)

	images, err := parseImageReferences(refs)
	require.NoError(t, err)
	require.Len(t, images, 2)

	assert.Equal(t, "etcd", images[0].Name)
	assert.Equal(t, "quay.io/openshift-release-dev/ocp-v4.0-art-dev@sha256:aaa", images[0].Image)
	assert.Equal(t, "sha256:aaa", images[0].Digest)

	assert.Equal(t, "cluster-network-operator", images[1].Name)
	assert.Equal(t, "sha256:bbb", images[1].Digest)
}

func TestParseImageReferencesEmpty(t *testing.T) {
	images, err := parseImageReferences([]byte(`{"spec":{"tags":[]}}`))
	require.NoError(t, err)
	assert.Empty(t, images)
}

func TestDigestFromRef(t *testing.T) {
	assert.Equal(t, "sha256:x", digestFromRef("repo@sha256:x"))
	assert.Equal(t, "", digestFromRef("repo:tag"))
}
