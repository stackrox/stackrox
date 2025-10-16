package image

import (
	"context"
	"fmt"
	"testing"

	"github.com/stackrox/rox/central/risk/multipliers"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/testutils"
)

func TestImageAgeScore(t *testing.T) {
	imageAgeMultiplier := NewImageAge()

	image := multipliers.GetMockImages()[0]
	rrf := &storage.Risk_Result_Factor{}
	rrf.SetMessage(fmt.Sprintf("Image %q is 180 days old", image.GetName().GetFullName()))
	expectedScore := &storage.Risk_Result{}
	expectedScore.SetName(ImageAgeHeading)
	expectedScore.SetFactors([]*storage.Risk_Result_Factor{
		rrf,
	})
	expectedScore.SetScore(1.25)

	score := imageAgeMultiplier.Score(context.Background(), image)
	protoassert.Equal(t, expectedScore, score)
}

func TestImageAgeScoreV2(t *testing.T) {
	testutils.MustUpdateFeature(t, features.FlattenImageData, true)
	imageAgeMultiplier := NewImageAge()

	image := multipliers.GetMockImagesV2()[0]
	rrf := &storage.Risk_Result_Factor{}
	rrf.SetMessage(fmt.Sprintf("Image %q is 180 days old", image.GetName().GetFullName()))
	expectedScore := &storage.Risk_Result{}
	expectedScore.SetName(ImageAgeHeading)
	expectedScore.SetFactors([]*storage.Risk_Result_Factor{
		rrf,
	})
	expectedScore.SetScore(1.25)

	score := imageAgeMultiplier.ScoreV2(context.Background(), image)
	protoassert.Equal(t, expectedScore, score)
}
