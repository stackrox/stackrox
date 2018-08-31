package clusters

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateCollectorImage(t *testing.T) {
	var cases = []struct {
		preventImage  string
		collectorTag  string
		expectedImage string
	}{
		{
			preventImage:  "stackrox/prevent:latest",
			collectorTag:  "latest",
			expectedImage: "docker.io/stackrox/collector:latest",
		},
		{
			preventImage:  "docker.io/stackrox/prevent:latest",
			collectorTag:  "latest",
			expectedImage: "docker.io/stackrox/collector:latest",
		},
		{
			preventImage:  "stackrox.io/prevent:latest",
			collectorTag:  "latest",
			expectedImage: "collector.stackrox.io/collector:latest",
		},
		{
			preventImage:  "stackrox.io/prevent:latest",
			collectorTag:  "loooool",
			expectedImage: "collector.stackrox.io/collector:loooool",
		},
	}

	for _, c := range cases {
		t.Run(c.preventImage, func(t *testing.T) {
			assert.Equal(t, c.expectedImage, generateCollectorImage(c.preventImage, c.collectorTag))
		})
	}
}
