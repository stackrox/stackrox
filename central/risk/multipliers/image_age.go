package multipliers

import (
	"context"
	"fmt"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/risk"
)

const (
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
func (c *imageAgeMultiplier) Score(_ context.Context, msg proto.Message) *storage.Risk_Result {
	image, ok := msg.(*storage.Image)
	if !ok {
		return nil
	}

	// Get the earliest created time in the container images, and find the duration since then.
	imageCreated := image.GetMetadata().GetV1().GetCreated()
	createdTime := protoconv.ConvertTimestampToTimeOrDefault(imageCreated, defaultTime)

	if createdTime.IsZero() {
		return nil
	}

	// Calculate days from creation time.
	durationSinceImageCreated := time.Since(createdTime)
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

	var message string
	if image.GetName().GetFullName() == "" {
		message = fmt.Sprintf("An image is %d days old", daysSinceCreated)
	} else {
		message = fmt.Sprintf("Image %q is %d days old", image.GetName().GetFullName(), daysSinceCreated)
	}

	return &storage.Risk_Result{
		Name: risk.ImageAge.DisplayTitle,
		Factors: []*storage.Risk_Result_Factor{
			{Message: message},
		},
		Score: score,
	}
}
