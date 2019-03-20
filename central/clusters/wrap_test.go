package clusters

import (
	"testing"

	"github.com/stackrox/rox/pkg/images/utils"
	"github.com/stretchr/testify/assert"
)

func TestGenerateCollectorImage(t *testing.T) {
	var cases = []struct {
		mainImage     string
		collectorTag  string
		expectedImage string
	}{
		{
			mainImage:     "stackrox/main:latest",
			collectorTag:  "latest",
			expectedImage: "docker.io/stackrox/collector:latest",
		},
		{
			mainImage:     "docker.io/stackrox/main:latest",
			collectorTag:  "latest",
			expectedImage: "docker.io/stackrox/collector:latest",
		},
		{
			mainImage:     "stackrox.io/main:latest",
			collectorTag:  "latest",
			expectedImage: "collector.stackrox.io/collector:latest",
		},
		{
			mainImage:     "stackrox.io/main:latest",
			collectorTag:  "loooool",
			expectedImage: "collector.stackrox.io/collector:loooool",
		},
	}

	for _, c := range cases {
		t.Run(c.mainImage, func(t *testing.T) {
			inputImg := utils.GenerateImageFromStringIgnoringError(c.mainImage)
			outputImg := utils.GenerateImageFromStringIgnoringError(c.expectedImage)
			assert.Equal(t, outputImg.GetName(), generateCollectorImage(inputImg.GetName(), c.collectorTag))
		})
	}
}
