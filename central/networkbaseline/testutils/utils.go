package testutils

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/networkgraph"
)

func GetBaselineWithCustomDeploymentFlow(
	deploymentName string,
	entityID, entityClusterID string,
	flowIsIngress bool,
	flowPort uint32,
) *storage.NetworkBaseline {
	baseline := fixtures.GetNetworkBaseline()
	baseline.Peers = []*storage.NetworkBaselinePeer{
		{
			Entity: &storage.NetworkEntity{
				Info: &storage.NetworkEntityInfo{
					Type: storage.NetworkEntityInfo_DEPLOYMENT,
					Id:   entityID,
					Desc: &storage.NetworkEntityInfo_Deployment_{
						Deployment: &storage.NetworkEntityInfo_Deployment{
							Name: deploymentName,
						},
					},
				},
				Scope: &storage.NetworkEntity_Scope{ClusterId: entityClusterID},
			},
			Properties: []*storage.NetworkBaselineConnectionProperties{
				{
					Ingress:  flowIsIngress,
					Port:     flowPort,
					Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
				},
			},
		},
	}

	return baseline
}

func GetBaselineWithInternet(
	entityClusterID string,
	flowIsIngress bool,
	flowPort uint32,
) *storage.NetworkBaseline {
	baseline := fixtures.GetNetworkBaseline()
	baseline.Peers = []*storage.NetworkBaselinePeer{
		{
			Entity: &storage.NetworkEntity{
				Info: &storage.NetworkEntityInfo{
					Type: storage.NetworkEntityInfo_INTERNET,
					Id:   networkgraph.InternetExternalSourceID,
				},
				Scope: &storage.NetworkEntity_Scope{ClusterId: entityClusterID},
			},
			Properties: []*storage.NetworkBaselineConnectionProperties{
				{
					Ingress:  flowIsIngress,
					Port:     flowPort,
					Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
				},
			},
		},
	}

	return baseline
}
