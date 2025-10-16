package networkbaseline

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/networkgraph"
	"github.com/stretchr/testify/assert"
)

func TestNetworkBaselineEvaluator(t *testing.T) {

	nei := &storage.NetworkEntityInfo{}
	nei.SetType(storage.NetworkEntityInfo_DEPLOYMENT)
	nei.SetId("dp1")
	nei2 := &storage.NetworkEntityInfo{}
	nei2.SetType(storage.NetworkEntityInfo_DEPLOYMENT)
	nei2.SetId("dp2")
	nfp := &storage.NetworkFlowProperties{}
	nfp.SetSrcEntity(nei)
	nfp.SetDstEntity(nei2)
	nfp.SetDstPort(80)
	nfp.SetL4Protocol(storage.L4Protocol_L4_PROTOCOL_TCP)
	inBaselineFlow := &storage.NetworkFlow{}
	inBaselineFlow.SetProps(nfp)

	nei3 := &storage.NetworkEntityInfo{}
	nei3.SetType(storage.NetworkEntityInfo_DEPLOYMENT)
	nei3.SetId("dp1")
	nei4 := &storage.NetworkEntityInfo{}
	nei4.SetType(storage.NetworkEntityInfo_DEPLOYMENT)
	nei4.SetId("dp3")
	nfp2 := &storage.NetworkFlowProperties{}
	nfp2.SetSrcEntity(nei3)
	nfp2.SetDstEntity(nei4)
	nfp2.SetDstPort(80)
	nfp2.SetL4Protocol(storage.L4Protocol_L4_PROTOCOL_TCP)
	anomalousFlow := &storage.NetworkFlow{}
	anomalousFlow.SetProps(nfp2)

	baseline := storage.NetworkBaseline_builder{
		DeploymentId: "dp1",
		ClusterId:    "cluster1",
		Namespace:    "namespace1",
		Peers: []*storage.NetworkBaselinePeer{
			storage.NetworkBaselinePeer_builder{
				Entity: storage.NetworkEntity_builder{
					Info: storage.NetworkEntityInfo_builder{
						Type: storage.NetworkEntityInfo_DEPLOYMENT,
						Id:   "dp2",
						Deployment: storage.NetworkEntityInfo_Deployment_builder{
							Name: "dp2",
						}.Build(),
					}.Build(),
				}.Build(),
				Properties: []*storage.NetworkBaselineConnectionProperties{
					storage.NetworkBaselineConnectionProperties_builder{
						Ingress:  false,
						Port:     80,
						Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
					}.Build(),
				},
			}.Build(),
		},
		ForbiddenPeers:       nil,
		ObservationPeriodEnd: nil,
		Locked:               false,
		DeploymentName:       "dp1",
	}.Build()

	evaluator := NewNetworkBaselineEvaluator()

	// No baseline yet. No flow should be outside of baseline
	assert.False(t, evaluator.IsOutsideLockedBaseline(inBaselineFlow, "dp1", "dp2"))
	assert.False(t, evaluator.IsOutsideLockedBaseline(anomalousFlow, "dp1", "dp3"))

	// Add an unlocked baseline, should have no effect
	assert.Nil(t, evaluator.AddBaseline(baseline))
	assert.False(t, evaluator.IsOutsideLockedBaseline(inBaselineFlow, "dp1", "dp2"))
	assert.False(t, evaluator.IsOutsideLockedBaseline(anomalousFlow, "dp1", "dp3"))

	// Add a locked baseline and check flow statuses
	baseline.SetLocked(true)
	assert.Nil(t, evaluator.AddBaseline(baseline))
	assert.False(t, evaluator.IsOutsideLockedBaseline(inBaselineFlow, "dp1", "dp2"))
	assert.True(t, evaluator.IsOutsideLockedBaseline(anomalousFlow, "dp1", "dp3"))

	// Remove the baseline should silent the above difference in flow statuses
	evaluator.RemoveBaselineByDeploymentID(baseline.GetDeploymentId())
	assert.False(t, evaluator.IsOutsideLockedBaseline(inBaselineFlow, "dp1", "dp2"))
	assert.False(t, evaluator.IsOutsideLockedBaseline(anomalousFlow, "dp1", "dp3"))

}

func TestDiscoveredExternalInBaseline(t *testing.T) {
	baseline := storage.NetworkBaseline_builder{
		DeploymentId: "dp1",
		ClusterId:    "cluster1",
		Namespace:    "namespace1",
		Peers: []*storage.NetworkBaselinePeer{
			storage.NetworkBaselinePeer_builder{
				Entity: storage.NetworkEntity_builder{
					Info: storage.NetworkEntityInfo_builder{
						Type: storage.NetworkEntityInfo_INTERNET,
						Id:   networkgraph.InternetExternalSourceID,
						ExternalSource: storage.NetworkEntityInfo_ExternalSource_builder{
							Name: networkgraph.InternetExternalSourceName,
						}.Build(),
					}.Build(),
				}.Build(),
				Properties: []*storage.NetworkBaselineConnectionProperties{
					storage.NetworkBaselineConnectionProperties_builder{
						Ingress:  false,
						Port:     80,
						Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
					}.Build(),
				},
			}.Build(),
		},
		ForbiddenPeers:       nil,
		ObservationPeriodEnd: nil,
		Locked:               false,
		DeploymentName:       "dp1",
	}.Build()

	inBaselineDiscoveredFlow := storage.NetworkFlow_builder{
		Props: storage.NetworkFlowProperties_builder{
			SrcEntity: storage.NetworkEntityInfo_builder{
				Type: storage.NetworkEntityInfo_DEPLOYMENT,
				Id:   "dp1",
			}.Build(),
			DstEntity: storage.NetworkEntityInfo_builder{
				Type: storage.NetworkEntityInfo_EXTERNAL_SOURCE,
				Id:   "ip1",
				ExternalSource: storage.NetworkEntityInfo_ExternalSource_builder{
					Discovered: true,
					Name:       "1.1.1.1",
				}.Build(),
			}.Build(),
			DstPort:    80,
			L4Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
		}.Build(),
	}.Build()

	anomalousDiscoveredFlow := storage.NetworkFlow_builder{
		Props: storage.NetworkFlowProperties_builder{
			SrcEntity: storage.NetworkEntityInfo_builder{
				Type: storage.NetworkEntityInfo_DEPLOYMENT,
				Id:   "dp1",
			}.Build(),
			DstEntity: storage.NetworkEntityInfo_builder{
				Type: storage.NetworkEntityInfo_EXTERNAL_SOURCE,
				Id:   "ip1",
				ExternalSource: storage.NetworkEntityInfo_ExternalSource_builder{
					Discovered: true,
					Name:       "1.1.1.1",
				}.Build(),
			}.Build(),
			DstPort:    1337,
			L4Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
		}.Build(),
	}.Build()

	evaluator := NewNetworkBaselineEvaluator()

	// No baseline yet. No flow should be outside of baseline
	assert.False(t, evaluator.IsOutsideLockedBaseline(inBaselineDiscoveredFlow, "dp1", "ip1"))
	assert.False(t, evaluator.IsOutsideLockedBaseline(anomalousDiscoveredFlow, "dp1", "ip1"))

	// Add an unlocked baseline, should have no effect
	assert.Nil(t, evaluator.AddBaseline(baseline))
	assert.False(t, evaluator.IsOutsideLockedBaseline(inBaselineDiscoveredFlow, "dp1", "ip1"))
	assert.False(t, evaluator.IsOutsideLockedBaseline(anomalousDiscoveredFlow, "dp1", "ip1"))

	// Add a locked baseline and check flow statuses
	baseline.SetLocked(true)
	assert.Nil(t, evaluator.AddBaseline(baseline))
	assert.False(t, evaluator.IsOutsideLockedBaseline(inBaselineDiscoveredFlow, "dp1", "ip1"))
	assert.True(t, evaluator.IsOutsideLockedBaseline(anomalousDiscoveredFlow, "dp1", "ip1"))

	// Remove the baseline should silent the above difference in flow statuses
	evaluator.RemoveBaselineByDeploymentID(baseline.GetDeploymentId())
	assert.False(t, evaluator.IsOutsideLockedBaseline(inBaselineDiscoveredFlow, "dp1", "ip1"))
	assert.False(t, evaluator.IsOutsideLockedBaseline(anomalousDiscoveredFlow, "dp1", "ip1"))
}
