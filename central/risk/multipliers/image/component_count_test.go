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
		eisc := &storage.EmbeddedImageScanComponent{}
		eisc.SetName(strconv.Itoa(i))
		eisc.SetVersion("1.0")
		components = append(components, eisc)
	}
	image.GetScan().SetComponents(components)

	rrf := &storage.Risk_Result_Factor{}
	rrf.SetMessage("Image \"docker.io/library/nginx:1.10\" contains 15 components")
	expectedScore := &storage.Risk_Result{}
	expectedScore.SetName(ComponentCountHeading)
	expectedScore.SetFactors([]*storage.Risk_Result_Factor{
		rrf,
	})
	expectedScore.SetScore(1.25)
	score := countMultiplier.Score(context.Background(), image)
	protoassert.Equal(t, expectedScore, score)
}

func TestComponentCountScoreV2(t *testing.T) {
	testutils.MustUpdateFeature(t, features.FlattenImageData, true)
	countMultiplier := NewComponentCount()

	// The image already has one unique component. So we need 14 components added for the count to be 15.
	image := multipliers.GetMockImagesV2()[0]
	components := image.GetScan().GetComponents()
	for i := 0; i < 14; i++ {
		eisc := &storage.EmbeddedImageScanComponent{}
		eisc.SetName(strconv.Itoa(i))
		eisc.SetVersion("1.0")
		components = append(components, eisc)
	}
	image.GetScan().SetComponents(components)

	rrf := &storage.Risk_Result_Factor{}
	rrf.SetMessage("Image \"docker.io/library/nginx:1.10\" contains 15 components")
	expectedScore := &storage.Risk_Result{}
	expectedScore.SetName(ComponentCountHeading)
	expectedScore.SetFactors([]*storage.Risk_Result_Factor{
		rrf,
	})
	expectedScore.SetScore(1.25)
	score := countMultiplier.ScoreV2(context.Background(), image)
	protoassert.Equal(t, expectedScore, score)

	// Add a component with same name as an existing component but different version
	eisc := &storage.EmbeddedImageScanComponent{}
	eisc.SetName("1")
	eisc.SetVersion("2.0")
	image.GetScan().SetComponents(append(image.GetScan().GetComponents(), eisc))
	// New component should be counted in the score
	score = countMultiplier.ScoreV2(context.Background(), image)
	rrf2 := &storage.Risk_Result_Factor{}
	rrf2.SetMessage("Image \"docker.io/library/nginx:1.10\" contains 16 components")
	expectedScore = &storage.Risk_Result{}
	expectedScore.SetName(ComponentCountHeading)
	expectedScore.SetFactors([]*storage.Risk_Result_Factor{
		rrf2,
	})
	expectedScore.SetScore(1.3)
	protoassert.Equal(t, expectedScore, score)

	// Less components than the floor (10) should return nil score
	image.GetScan().SetComponents(image.GetScan().GetComponents()[:10])
	score = countMultiplier.ScoreV2(context.Background(), image)
	protoassert.Equal(t, nil, score)

	// More components than the ceiling (20) should return the max score
	for i := 14; i < 20; i++ {
		eisc2 := &storage.EmbeddedImageScanComponent{}
		eisc2.SetName(strconv.Itoa(i))
		eisc2.SetVersion("1.0")
		components = append(components, eisc2)
	}
	image.GetScan().SetComponents(components)
	rrf3 := &storage.Risk_Result_Factor{}
	rrf3.SetMessage("Image \"docker.io/library/nginx:1.10\" contains 21 components")
	expectedScore = &storage.Risk_Result{}
	expectedScore.SetName(ComponentCountHeading)
	expectedScore.SetFactors([]*storage.Risk_Result_Factor{
		rrf3,
	})
	expectedScore.SetScore(1.5)
	score = countMultiplier.ScoreV2(context.Background(), image)
	protoassert.Equal(t, expectedScore, score)
}
