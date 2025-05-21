package networkbaseline

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/networkgraph"
	"github.com/stretchr/testify/assert"
)

func TestNetworkBaselineEvaluator(t *testing.T) {

	inBaselineFlow := &storage.NetworkFlow{
		Props: &storage.NetworkFlowProperties{
			SrcEntity: &storage.NetworkEntityInfo{
				Type: storage.NetworkEntityInfo_DEPLOYMENT,
				Id:   "dp1",
			},
			DstEntity: &storage.NetworkEntityInfo{
				Type: storage.NetworkEntityInfo_DEPLOYMENT,
				Id:   "dp2",
			},
			DstPort:    80,
			L4Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
		},
	}

	anomalousFlow := &storage.NetworkFlow{
		Props: &storage.NetworkFlowProperties{
			SrcEntity: &storage.NetworkEntityInfo{
				Type: storage.NetworkEntityInfo_DEPLOYMENT,
				Id:   "dp1",
			},
			DstEntity: &storage.NetworkEntityInfo{
				Type: storage.NetworkEntityInfo_DEPLOYMENT,
				Id:   "dp3",
			},
			DstPort:    80,
			L4Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
		},
	}

	baseline := &storage.NetworkBaseline{
		DeploymentId: "dp1",
		ClusterId:    "cluster1",
		Namespace:    "namespace1",
		Peers: []*storage.NetworkBaselinePeer{
			{
				Entity: &storage.NetworkEntity{
					Info: &storage.NetworkEntityInfo{
						Type: storage.NetworkEntityInfo_DEPLOYMENT,
						Id:   "dp2",
						Desc: &storage.NetworkEntityInfo_Deployment_{
							Deployment: &storage.NetworkEntityInfo_Deployment{
								Name: "dp2",
							},
						},
					},
				},
				Properties: []*storage.NetworkBaselineConnectionProperties{
					{
						Ingress:  false,
						Port:     80,
						Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
					},
				},
			},
		},
		ForbiddenPeers:       nil,
		ObservationPeriodEnd: nil,
		Locked:               false,
		DeploymentName:       "dp1",
	}

	evaluator := NewNetworkBaselineEvaluator()

	// No baseline yet. No flow should be outside of baseline
	assert.False(t, evaluator.IsOutsideLockedBaseline(inBaselineFlow, "dp1", "dp2"))
	assert.False(t, evaluator.IsOutsideLockedBaseline(anomalousFlow, "dp1", "dp3"))

	// Add an unlocked baseline, should have no effect
	assert.Nil(t, evaluator.AddBaseline(baseline))
	assert.False(t, evaluator.IsOutsideLockedBaseline(inBaselineFlow, "dp1", "dp2"))
	assert.False(t, evaluator.IsOutsideLockedBaseline(anomalousFlow, "dp1", "dp3"))

	// Add a locked baseline and check flow statuses
	baseline.Locked = true
	assert.Nil(t, evaluator.AddBaseline(baseline))
	assert.False(t, evaluator.IsOutsideLockedBaseline(inBaselineFlow, "dp1", "dp2"))
	assert.True(t, evaluator.IsOutsideLockedBaseline(anomalousFlow, "dp1", "dp3"))

	// Remove the baseline should silent the above difference in flow statuses
	evaluator.RemoveBaselineByDeploymentID(baseline.GetDeploymentId())
	assert.False(t, evaluator.IsOutsideLockedBaseline(inBaselineFlow, "dp1", "dp2"))
	assert.False(t, evaluator.IsOutsideLockedBaseline(anomalousFlow, "dp1", "dp3"))

}

func TestDiscoveredExternalInBaseline(t *testing.T) {
	baseline := &storage.NetworkBaseline{
		DeploymentId: "dp1",
		ClusterId:    "cluster1",
		Namespace:    "namespace1",
		Peers: []*storage.NetworkBaselinePeer{
			{
				Entity: &storage.NetworkEntity{
					Info: &storage.NetworkEntityInfo{
						Type: storage.NetworkEntityInfo_INTERNET,
						Id:   networkgraph.InternetExternalSourceID,
						Desc: &storage.NetworkEntityInfo_ExternalSource_{
							ExternalSource: &storage.NetworkEntityInfo_ExternalSource{
								Name: networkgraph.InternetExternalSourceName,
							},
						},
					},
				},
				Properties: []*storage.NetworkBaselineConnectionProperties{
					{
						Ingress:  false,
						Port:     80,
						Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
					},
				},
			},
		},
		ForbiddenPeers:       nil,
		ObservationPeriodEnd: nil,
		Locked:               false,
		DeploymentName:       "dp1",
	}

	inBaselineDiscoveredFlow := &storage.NetworkFlow{
		Props: &storage.NetworkFlowProperties{
			SrcEntity: &storage.NetworkEntityInfo{
				Type: storage.NetworkEntityInfo_DEPLOYMENT,
				Id:   "dp1",
			},
			DstEntity: &storage.NetworkEntityInfo{
				Type: storage.NetworkEntityInfo_EXTERNAL_SOURCE,
				Id:   "ip1",
				Desc: &storage.NetworkEntityInfo_ExternalSource_{
					ExternalSource: &storage.NetworkEntityInfo_ExternalSource{
						Discovered: true,
						Name:       "1.1.1.1",
					},
				},
			},
			DstPort:    80,
			L4Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
		},
	}

	anomalousDiscoveredFlow := &storage.NetworkFlow{
		Props: &storage.NetworkFlowProperties{
			SrcEntity: &storage.NetworkEntityInfo{
				Type: storage.NetworkEntityInfo_DEPLOYMENT,
				Id:   "dp1",
			},
			DstEntity: &storage.NetworkEntityInfo{
				Type: storage.NetworkEntityInfo_EXTERNAL_SOURCE,
				Id:   "ip1",
				Desc: &storage.NetworkEntityInfo_ExternalSource_{
					ExternalSource: &storage.NetworkEntityInfo_ExternalSource{
						Discovered: true,
						Name:       "1.1.1.1",
					},
				},
			},
			DstPort:    1337,
			L4Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
		},
	}

	evaluator := NewNetworkBaselineEvaluator()

	// No baseline yet. No flow should be outside of baseline
	assert.False(t, evaluator.IsOutsideLockedBaseline(inBaselineDiscoveredFlow, "dp1", "ip1"))
	assert.False(t, evaluator.IsOutsideLockedBaseline(anomalousDiscoveredFlow, "dp1", "ip1"))

	// Add an unlocked baseline, should have no effect
	assert.Nil(t, evaluator.AddBaseline(baseline))
	assert.False(t, evaluator.IsOutsideLockedBaseline(inBaselineDiscoveredFlow, "dp1", "ip1"))
	assert.False(t, evaluator.IsOutsideLockedBaseline(anomalousDiscoveredFlow, "dp1", "ip1"))

	// Add a locked baseline and check flow statuses
	baseline.Locked = true
	assert.Nil(t, evaluator.AddBaseline(baseline))
	assert.False(t, evaluator.IsOutsideLockedBaseline(inBaselineDiscoveredFlow, "dp1", "ip1"))
	assert.True(t, evaluator.IsOutsideLockedBaseline(anomalousDiscoveredFlow, "dp1", "ip1"))

	// Remove the baseline should silent the above difference in flow statuses
	evaluator.RemoveBaselineByDeploymentID(baseline.GetDeploymentId())
	assert.False(t, evaluator.IsOutsideLockedBaseline(inBaselineDiscoveredFlow, "dp1", "ip1"))
	assert.False(t, evaluator.IsOutsideLockedBaseline(anomalousDiscoveredFlow, "dp1", "ip1"))
}
