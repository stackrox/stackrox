package resources

import (
	"testing"

	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestParseContainerID(t *testing.T) {
	cases := []struct {
		input           string
		expectedRuntime storage.ContainerRuntime
		expectedID      string
	}{
		{
			input:           "cri-o://6f6b9de202f6ec9939bcbecf0beebe1f56c6f521922d4073467136b536c75f53",
			expectedRuntime: storage.ContainerRuntime_CRIO_CONTAINER_RUNTIME,
			expectedID:      "6f6b9de202f6ec9939bcbecf0beebe1f56c6f521922d4073467136b536c75f53",
		},
		{
			input:           "docker://f7c38c083c68bc76709b4f27830517d7676434b13a48e348021a62df8f966aaa",
			expectedRuntime: storage.ContainerRuntime_DOCKER_CONTAINER_RUNTIME,
			expectedID:      "f7c38c083c68bc76709b4f27830517d7676434b13a48e348021a62df8f966aaa",
		},
		{
			input:           "some-runtime://fad8a0abe03017917c868cd2e009827db8c7d64aa8a4959afd02f9a6e11daa06",
			expectedRuntime: storage.ContainerRuntime_UNKNOWN_CONTAINER_RUNTIME,
			expectedID:      "fad8a0abe03017917c868cd2e009827db8c7d64aa8a4959afd02f9a6e11daa06",
		},
		{
			input:           "cd1644c9dbb5ab1919ce34be8f59225771487853557aa5b2aa291ad6387abc80",
			expectedRuntime: storage.ContainerRuntime_UNKNOWN_CONTAINER_RUNTIME,
			expectedID:      "cd1644c9dbb5ab1919ce34be8f59225771487853557aa5b2aa291ad6387abc80",
		},
	}

	for _, tc := range cases {
		c := tc
		t.Run("Test with input "+c.input, func(t *testing.T) {
			runtime, id := parseContainerID(c.input)
			assert.Equal(t, c.expectedID, id)
			assert.Equal(t, c.expectedRuntime, runtime)
		})
	}
}
