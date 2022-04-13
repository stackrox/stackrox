package image

import (
	"context"
	"fmt"
	"time"

	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/protoconv"
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

// vulnerabilitiesMultiplier is a scorer for the vulnerabilities in an image
type imageAgeMultiplier struct{}

// NewImageAge provides a multiplier that scores the data based on the age of the image it contains.
func NewImageAge() Multiplier {
	return &imageAgeMultiplier{}
}

// Score takes a image and evaluates its risk based on age (days since creation)
func (c *imageAgeMultiplier) Score(_ context.Context, image *storage.Image) *storage.Risk_Result {
	imageCreated := image.GetMetadata().GetV1().GetCreated()
	createdTime := protoconv.ConvertTimestampToTimeOrDefault(imageCreated, defaultTime)
	if createdTime.IsZero() {
		return nil
	}

	// Calculate days from creation time.
	durationSinceImageCreated := time.Since(createdTime)
	daysSinceCreated := int(durationSinceImageCreated.Hours() / 24)
	score := GetImageAgeRiskScore(daysSinceCreated)
	if score == 0.0 {
		return nil
	}
	message := GetImageAgeRiskFactorMessage(image.GetName().GetFullName(), daysSinceCreated)

	return &storage.Risk_Result{
		Name: ImageAgeHeading,
		Factors: []*storage.Risk_Result_Factor{
			{Message: message},
		},
		Score: score,
	}
}

// GetImageAgeRiskScore returns a risk score for given image age.
func GetImageAgeRiskScore(imageAgeInDays int) (riskScore float32) {
	// Creates a score that is:
	// A) No risk when daysSinceCreated is < penalizeDaysFloor
	if imageAgeInDays < penalizeDaysFloor {
		return
	}

	// B) Increases linearly between penalizeDaysFloor and penalizeDaysCeil
	daysSincePenalized := imageAgeInDays - penalizeDaysFloor
	scaledDays := float32(daysSincePenalized) / float32(penalizeDaysCeil-penalizeDaysFloor)
	riskScore = float32(1) + float32(scaledDays*(maxMultiplier-1))

	// C) Is 1.5 when duration is > penalizeDaysCeil
	if riskScore > maxMultiplier {
		riskScore = maxMultiplier
	}
	return
}

// GetImageAgeRiskFactorMessage returns risk maessage for image age risk.
func GetImageAgeRiskFactorMessage(imageFullName string, imageAgeIndays int) (riskFactorMessage string) {
	if imageFullName == "" {
		riskFactorMessage = fmt.Sprintf("An image is %d days old", imageAgeIndays)
	} else {
		riskFactorMessage = fmt.Sprintf("Image %q is %d days old", imageFullName, imageAgeIndays)
	}
	return
}
