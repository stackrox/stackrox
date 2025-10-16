package image

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/risk/multipliers"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/testutils"
)

func TestRiskyComponentCountScore(t *testing.T) {
	riskyMultiplier := NewRiskyComponents()

	// Add some risky components to the deployment
	images := multipliers.GetMockImages()
	components := images[0].GetScan().GetComponents()
	eisc := &storage.EmbeddedImageScanComponent{}
	eisc.SetName("apk")
	eisc.SetVersion("1.0")
	components = append(components, eisc)
	eisc2 := &storage.EmbeddedImageScanComponent{}
	eisc2.SetName("apk")
	eisc2.SetVersion("1.2")
	components = append(components, eisc2)
	eisc3 := &storage.EmbeddedImageScanComponent{}
	eisc3.SetName("tcsh")
	eisc3.SetVersion("1.0")
	components = append(components, eisc3)
	eisc4 := &storage.EmbeddedImageScanComponent{}
	eisc4.SetName("curl")
	eisc4.SetVersion("1.0")
	components = append(components, eisc4)
	eisc5 := &storage.EmbeddedImageScanComponent{}
	eisc5.SetName("wget")
	eisc5.SetVersion("1.0")
	components = append(components, eisc5)
	eisc6 := &storage.EmbeddedImageScanComponent{}
	eisc6.SetName("telnet")
	eisc6.SetVersion("1.0")
	components = append(components, eisc6)
	eisc7 := &storage.EmbeddedImageScanComponent{}
	eisc7.SetName("yum")
	eisc7.SetVersion("1.0")
	components = append(components, eisc7)

	images[0].GetScan().SetComponents(components)

	rrf := &storage.Risk_Result_Factor{}
	rrf.SetMessage("Image \"docker.io/library/nginx:1.10\" contains components: apk, curl, tcsh, telnet, wget and 1 other(s) that are useful for attackers")
	expectedScore := &storage.Risk_Result{}
	expectedScore.SetName(RiskyComponentCountHeading)
	expectedScore.SetFactors([]*storage.Risk_Result_Factor{
		rrf,
	})
	expectedScore.SetScore(1.3)
	score := riskyMultiplier.Score(context.Background(), images[0])
	protoassert.Equal(t, expectedScore, score)
}

func TestRiskyComponentCountScoreV2(t *testing.T) {
	testutils.MustUpdateFeature(t, features.FlattenImageData, true)
	riskyMultiplier := NewRiskyComponents()

	// Add some risky components to the deployment
	images := multipliers.GetMockImagesV2()
	components := images[0].GetScan().GetComponents()
	eisc := &storage.EmbeddedImageScanComponent{}
	eisc.SetName("apk")
	eisc.SetVersion("1.0")
	components = append(components, eisc)
	eisc2 := &storage.EmbeddedImageScanComponent{}
	eisc2.SetName("apk")
	eisc2.SetVersion("1.2")
	components = append(components, eisc2)
	eisc3 := &storage.EmbeddedImageScanComponent{}
	eisc3.SetName("tcsh")
	eisc3.SetVersion("1.0")
	components = append(components, eisc3)
	eisc4 := &storage.EmbeddedImageScanComponent{}
	eisc4.SetName("curl")
	eisc4.SetVersion("1.0")
	components = append(components, eisc4)
	eisc5 := &storage.EmbeddedImageScanComponent{}
	eisc5.SetName("wget")
	eisc5.SetVersion("1.0")
	components = append(components, eisc5)
	eisc6 := &storage.EmbeddedImageScanComponent{}
	eisc6.SetName("telnet")
	eisc6.SetVersion("1.0")
	components = append(components, eisc6)
	eisc7 := &storage.EmbeddedImageScanComponent{}
	eisc7.SetName("yum")
	eisc7.SetVersion("1.0")
	components = append(components, eisc7)

	images[0].GetScan().SetComponents(components)

	rrf := &storage.Risk_Result_Factor{}
	rrf.SetMessage("Image \"docker.io/library/nginx:1.10\" contains components: apk, curl, tcsh, telnet, wget and 1 other(s) that are useful for attackers")
	expectedScore := &storage.Risk_Result{}
	expectedScore.SetName(RiskyComponentCountHeading)
	expectedScore.SetFactors([]*storage.Risk_Result_Factor{
		rrf,
	})
	expectedScore.SetScore(1.3)
	score := riskyMultiplier.ScoreV2(context.Background(), images[0])
	protoassert.Equal(t, expectedScore, score)
}
