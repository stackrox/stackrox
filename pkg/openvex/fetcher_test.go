package openvex

import (
	"context"
	"os"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/images/types"
	imgUtils "github.com/stackrox/rox/pkg/images/utils"
	registryTypes "github.com/stackrox/rox/pkg/registries/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
Sample VEX JSON for image docker.io/daha97/openvex:1.23.4:
Note the product ID reference, this is used to determin which image to attach the VEX report to via vexctl attach
{
  "@context": "https://openvex.dev/ns/v0.2.0",
  "@id": "https://openvex.dev/docs/public/vex-62acec5ff5ddcd0fbe8deb6f4a2a31fce0a441b8ead418bbf2ffd951ea94c321",
  "author": "dhaus",
  "timestamp": "2023-09-28T02:55:21.820199+02:00",
  "version": 1,
  "statements": [
    {
      "vulnerability": {
        "name": "CVE-2022-1304"
      },
      "timestamp": "2023-09-28T02:55:21.820202+02:00",
      "products": [
        {
          "@id": "pkg:oci/openvex?repository_url=docker.io/daha97&tag=1.23.4"
        }
      ],
      "status": "not_affected",
      "justification": "vulnerable_code_not_in_execute_path",
      "impact_statement": "nothing"
    }
  ]
}
*/

type testRegistry struct {
	registryTypes.Registry
	cfg *registryTypes.Config
}

func (t *testRegistry) Config() *registryTypes.Config {
	return t.cfg
}

func TestFetch(t *testing.T) {
	t.Skip("Skipping in CI, this is only used for local testing really")
	f := NewFetcher()
	r := &testRegistry{cfg: &registryTypes.Config{
		Username: os.Getenv("DOCKER_USERNAME"),
		Password: os.Getenv("DOCKER_PASSWORD"),
		URL:      "docker.io",
	}}
	cimg, err := imgUtils.GenerateImageFromString("docker.io/daha97/openvex:1.23.4")
	require.NoError(t, err)
	img := types.ToImage(cimg)
	img.Metadata = &storage.ImageMetadata{
		V2: &storage.V2Metadata{
			Digest: "asdasdsadas",
		},
	}

	vex, err := f.Fetch(context.Background(), img, r)
	assert.NoError(t, err)
	assert.NotEmpty(t, vex)
}
