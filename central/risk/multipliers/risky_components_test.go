package multipliers

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestRiskyComponentCountScore(t *testing.T) {
	riskyMultiplier := NewRiskyComponents()
	deployment := getMockDeployment()

	// Add some risky components to the deployment
	components := deployment.Containers[0].Image.Scan.Components
	components = append(components, &storage.ImageScanComponent{
		Name:    "apk",
		Version: "1.0",
	})
	components = append(components, &storage.ImageScanComponent{
		Name:    "apk",
		Version: "1.2",
	})
	components = append(components, &storage.ImageScanComponent{
		Name:    "tcsh",
		Version: "1.0",
	})
	components = append(components, &storage.ImageScanComponent{
		Name:    "curl",
		Version: "1.0",
	})
	components = append(components, &storage.ImageScanComponent{
		Name:    "wget",
		Version: "1.0",
	})
	components = append(components, &storage.ImageScanComponent{
		Name:    "telnet",
		Version: "1.0",
	})
	components = append(components, &storage.ImageScanComponent{
		Name:    "yum",
		Version: "1.0",
	})

	deployment.Containers[0].Image.Scan.Components = components

	expectedScore := &storage.Risk_Result{
		Name: RiskyComponentCountHeading,
		Factors: []*storage.Risk_Result_Factor{
			{Message: "An image contains components: apk, curl, tcsh, telnet, wget and 1 other(s) that are useful for attackers"},
		},
		Score: 1.3,
	}
	score := riskyMultiplier.Score(deployment)
	assert.Equal(t, expectedScore, score)
}
