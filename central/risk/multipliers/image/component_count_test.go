package image

import (
	"context"
	"strconv"
	"testing"

	"github.com/stackrox/rox/central/risk/multipliers"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/testutils"
)

func TestComponentCountScore(t *testing.T) {
	countMultiplier := NewComponentCount()

	// We need 14 components added for the count to be 15
	image := multipliers.GetMockImages()[0]
	components := image.GetScan().GetComponents()
	for i := 0; i < 14; i++ {
		components = append(components, &storage.EmbeddedImageScanComponent{
			Name:    strconv.Itoa(i),
			Version: "1.0",
		})
	}
	image.Scan.Components = components

	expectedScore := &storage.Risk_Result{
		Name: ComponentCountHeading,
		Factors: []*storage.Risk_Result_Factor{
			{Message: "Image \"docker.io/library/nginx:1.10\" contains 15 components"},
		},
		Score: 1.25,
	}
	score := countMultiplier.Score(context.Background(), image)
	protoassert.Equal(t, expectedScore, score)
}

func TestComponentCountScoreV2(t *testing.T) {
	testutils.MustUpdateFeature(t, features.FlattenImageData, true)
	countMultiplier := NewComponentCount()

	// We need 14 components added for the count to be 15
	image := multipliers.GetMockImagesV2()[0]
	components := image.GetScan().GetComponents()
	for i := 0; i < 14; i++ {
		components = append(components, &storage.EmbeddedImageScanComponent{
			Name:    strconv.Itoa(i),
			Version: "1.0",
		})
	}
	image.Scan.Components = components

	expectedScore := &storage.Risk_Result{
		Name: ComponentCountHeading,
		Factors: []*storage.Risk_Result_Factor{
			{Message: "Image \"docker.io/library/nginx:1.10\" contains 15 components"},
		},
		Score: 1.25,
	}
	score := countMultiplier.ScoreV2(context.Background(), image)
	protoassert.Equal(t, expectedScore, score)
}
