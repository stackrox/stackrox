package clusters

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateCollectorImage(t *testing.T) {
	var cases = []struct {
		preventImage  string
		expectedImage string
	}{
		{
			preventImage:  "stackrox/prevent:latest",
			expectedImage: "docker.io/stackrox/collector:latest",
		},
		{
			preventImage:  "docker.io/stackrox/prevent:latest",
			expectedImage: "docker.io/stackrox/collector:latest",
		},
		{
			preventImage:  "stackrox.io/prevent:latest",
			expectedImage: "collector.stackrox.io/collector:latest",
		},
	}

	for _, c := range cases {
		t.Run(c.preventImage, func(t *testing.T) {
			assert.Equal(t, c.expectedImage, generateCollectorImage(c.preventImage))
		})
	}
}
