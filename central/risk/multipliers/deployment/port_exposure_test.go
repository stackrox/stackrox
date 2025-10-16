package deployment

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/risk/multipliers"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoassert"
)

func TestPortExposureScore(t *testing.T) {
	portMultiplier := NewReachability()

	deployment := multipliers.GetMockDeployment()
	rrf := &storage.Risk_Result_Factor{}
	rrf.SetMessage("Port 22 is exposed to external clients")
	rrf2 := &storage.Risk_Result_Factor{}
	rrf2.SetMessage("Port 23 is exposed in the cluster")
	rrf3 := &storage.Risk_Result_Factor{}
	rrf3.SetMessage("Port 24 is exposed on node interfaces")
	expectedScore := &storage.Risk_Result{}
	expectedScore.SetName(ReachabilityHeading)
	expectedScore.SetFactors([]*storage.Risk_Result_Factor{
		rrf,
		rrf2,
		rrf3,
	})
	expectedScore.SetScore(1.6)
	score := portMultiplier.Score(context.Background(), deployment, nil)
	protoassert.Equal(t, expectedScore, score)
}
