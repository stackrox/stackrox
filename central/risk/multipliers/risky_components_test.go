package multipliers

import (
	"testing"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stretchr/testify/assert"
)

func TestRiskyComponentCountScore(t *testing.T) {
	portMultiplier := NewRiskyComponents()
	deployment := getMockDeployment()

	// Add some risky components to the deployment
	components := deployment.Containers[0].Image.Scan.Components
	components = append(components, &v1.ImageScanComponent{
		Name:    "apk",
		Version: "1.0",
	})
	components = append(components, &v1.ImageScanComponent{
		Name:    "apk",
		Version: "1.2",
	})
	components = append(components, &v1.ImageScanComponent{
		Name:    "tcsh",
		Version: "1.0",
	})
	components = append(components, &v1.ImageScanComponent{
		Name:    "curl",
		Version: "1.0",
	})
	components = append(components, &v1.ImageScanComponent{
		Name:    "wget",
		Version: "1.0",
	})
	components = append(components, &v1.ImageScanComponent{
		Name:    "telnet",
		Version: "1.0",
	})
	components = append(components, &v1.ImageScanComponent{
		Name:    "yum",
		Version: "1.0",
	})

	deployment.Containers[0].Image.Scan.Components = components

	expectedScore := &v1.Risk_Result{
		Name: RiskyComponentCountHeading,
		Factors: []*v1.Risk_Result_Factor{
			{Message: "an image contains components: apk, curl, tcsh, telnet, wget and 1 other(s) that are useful for attackers"},
		},
		Score: 1.3,
	}
	score := portMultiplier.Score(deployment)
	assert.Equal(t, expectedScore, score)
}
