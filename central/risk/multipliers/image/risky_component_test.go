package image

import (
	"context"
	"testing"

	"github.com/stackrox/stackrox/central/risk/multipliers"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestRiskyComponentCountScore(t *testing.T) {
	riskyMultiplier := NewRiskyComponents()

	// Add some risky components to the deployment
	images := multipliers.GetMockImages()
	components := images[0].Scan.Components
	components = append(components, &storage.EmbeddedImageScanComponent{
		Name:    "apk",
		Version: "1.0",
	})
	components = append(components, &storage.EmbeddedImageScanComponent{
		Name:    "apk",
		Version: "1.2",
	})
	components = append(components, &storage.EmbeddedImageScanComponent{
		Name:    "tcsh",
		Version: "1.0",
	})
	components = append(components, &storage.EmbeddedImageScanComponent{
		Name:    "curl",
		Version: "1.0",
	})
	components = append(components, &storage.EmbeddedImageScanComponent{
		Name:    "wget",
		Version: "1.0",
	})
	components = append(components, &storage.EmbeddedImageScanComponent{
		Name:    "telnet",
		Version: "1.0",
	})
	components = append(components, &storage.EmbeddedImageScanComponent{
		Name:    "yum",
		Version: "1.0",
	})

	images[0].Scan.Components = components

	expectedScore := &storage.Risk_Result{
		Name: RiskyComponentCountHeading,
		Factors: []*storage.Risk_Result_Factor{
			{Message: "Image \"docker.io/library/nginx:1.10\" contains components: apk, curl, tcsh, telnet, wget and 1 other(s) that are useful for attackers"},
		},
		Score: 1.3,
	}
	score := riskyMultiplier.Score(context.Background(), images[0])
	assert.Equal(t, expectedScore, score)
}
