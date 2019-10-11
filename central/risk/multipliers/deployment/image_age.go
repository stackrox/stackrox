package deployment

import (
	"context"
	"time"

	imageMultiplier "github.com/stackrox/rox/central/risk/multipliers/image"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoconv"
)

var defaultTime time.Time

// vulnerabilitiesMultiplier is a scorer for the vulnerabilities in an image
type imageAgeMultiplier struct {
	imageScorer imageMultiplier.Multiplier
}

// NewImageAge provides a multiplier that scores the data based on the age of the image it contains.
func NewImageAge() Multiplier {
	return &imageAgeMultiplier{
		imageScorer: imageMultiplier.NewImageAge(),
	}
}

// Score takes a deployment's images and evaluates its risk based on age (days since creation)
func (c *imageAgeMultiplier) Score(_ context.Context, _ *storage.Deployment, images []*storage.Image) *storage.Risk_Result {
	// Get the earliest created time in the container images, and find the duration since then.
	return c.imageScorer.Score(allAccessCtx, getOldestImage(images))
}

// Fetches the oldest image based on creation time.
func getOldestImage(images []*storage.Image) *storage.Image {
	var earliest time.Time
	var oldestImage *storage.Image

	for _, img := range images {
		// Get the time for the containers image.
		imageCreated := img.GetMetadata().GetV1().GetCreated()

		createdTime := protoconv.ConvertTimestampToTimeOrDefault(imageCreated, defaultTime)
		if createdTime == defaultTime {
			continue
		}

		// Check against max.
		if earliest.IsZero() || createdTime.Before(earliest) {
			earliest = createdTime
			oldestImage = img
		}
	}
	return oldestImage
}
