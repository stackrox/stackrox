package nexus

import (
	"os"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/registries/types"
	"github.com/stackrox/rox/pkg/urlfmt"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

// This requires a Nexus Sonatype registry with a proxy to Dockerhub
func TestNexus(t *testing.T) {
	t.Setenv("ROX_REGISTRY_RESPONSE_TIMEOUT", "90s")
	t.Setenv("ROX_REGISTRY_CLIENT_TIMEOUT", "120s")

	endpoint := os.Getenv("NEXUS_ENDPOINT")
	if endpoint == "" {
		t.Skipf("ENDPOINT is required for Nexus integration test")
	}

	typ, creator := Creator()
	require.Equal(t, types.NexusType, typ)

	dc := &storage.DockerConfig{}
	dc.SetEndpoint(endpoint)
	dc.SetUsername("admin")
	dc.SetPassword("admin123")
	ii := &storage.ImageIntegration{}
	ii.SetDocker(proto.ValueOrDefault(dc))
	reg, err := creator(ii)
	require.NoError(t, err)

	endpoint = urlfmt.TrimHTTPPrefixes(endpoint)
	imageName := &storage.ImageName{}
	imageName.SetRegistry(endpoint)
	imageName.SetRemote("nginx")
	imageName.SetTag("latest")
	image := &storage.Image{}
	image.SetId("sha256:e2847e35d4e0e2d459a7696538cbfea42ea2d3b8a1ee8329ba7e68694950afd3")
	image.SetName(imageName)
	_, err = reg.Metadata(&image)
	require.NoError(t, err)
}
