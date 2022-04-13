package docker

import (
	"testing"

	"github.com/stackrox/stackrox/pkg/docker"
	"github.com/stretchr/testify/require"
)

func BenchmarkContainerFetch(b *testing.B) {
	client, err := docker.NewClient()
	require.NoError(b, err)

	for i := 0; i < b.N; i++ {
		_, err := getContainers(client)
		require.NoError(b, err)
	}
}

func BenchmarkImageFetch(b *testing.B) {
	client, err := docker.NewClient()
	require.NoError(b, err)

	for i := 0; i < b.N; i++ {
		_, err := getImages(client)
		require.NoError(b, err)
	}
}
