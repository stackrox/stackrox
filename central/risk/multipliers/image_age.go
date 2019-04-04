package multipliers

import (
	"fmt"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoconv"
)

const (
	// ImageAgeHeading is the risk result name for scores calculated by this multiplier.
	ImageAgeHeading = "Image Freshness"

	// You can tune the multiplier behavior from here:
	// Minimum staleness before we add risk.
	penalizeDaysFloor = 90
	// At this staleness risk is maxed out.
	penalizeDaysCeil = 270
	// Maximum risk multiplier we can have.
	maxMultiplier = float32(1.5)
)

var defaultTime time.Time

// vulnerabilitiesMultiplier is a scorer for the vulnerabilities in a deployment
type imageAgeMultiplier struct{}

// NewImageAge provides a multiplier that scores the data based on the age of the images it contains.
func NewImageAge() Multiplier {
	return &imageAgeMultiplier{}
}

// Score takes a deployment and evaluates its risk based on vulnerabilties
func (c *imageAgeMultiplier) Score(deployment *storage.Deployment) *storage.Risk_Result {
	// Get the earliest created time in the container images, and find the duration since then.
	earliestImageCreated := getOldestCreatedTime(deployment)
	if earliestImageCreated.IsZero() {
		return nil
	}

	// Calculate days from creation time.
	durationSinceImageCreated := time.Since(earliestImageCreated)
	daysSinceCreated := int(durationSinceImageCreated.Hours() / 24)

	// Creates a score that is:
	// A) No risk when daysSinceCreated is < penalizeDaysFloor
	if daysSinceCreated < penalizeDaysFloor {
		return nil
	}

	// B) Increases linearly between penalizeDaysFloor and penalizeDaysCeil
	daysSincePenalized := daysSinceCreated - penalizeDaysFloor
	scaledDays := float32(daysSincePenalized) / float32(penalizeDaysCeil-penalizeDaysFloor)
	score := float32(1) + float32(scaledDays*(maxMultiplier-1))

	// C) Is 1.5 when duration is > penalizeDaysCeil
	if score > maxMultiplier {
		score = maxMultiplier
	}

	return &storage.Risk_Result{
		Name: ImageAgeHeading,
		Factors: []*storage.Risk_Result_Factor{
			{Message: fmt.Sprintf("Deployment contains an image %d days old", daysSinceCreated)},
		},
		Score: score,
	}
}

// Fetches the creation time of the oldest image in the deployment.
func getOldestCreatedTime(deployment *storage.Deployment) time.Time {
	var earliest time.Time
	for _, container := range deployment.GetContainers() {
		// Get the time for the containers image.
		imageCreated := container.GetImage().GetMetadata().GetV1().GetCreated()

		createdTime := protoconv.ConvertTimestampToTimeOrDefault(imageCreated, defaultTime)
		if createdTime == defaultTime {
			continue
		}

		// Check against max.
		if earliest.IsZero() || createdTime.Before(earliest) {
			earliest = createdTime
		}
	}
	return earliest
}
