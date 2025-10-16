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
	baseline.SetPeers([]*storage.NetworkBaselinePeer{
		storage.NetworkBaselinePeer_builder{
			Entity: storage.NetworkEntity_builder{
				Info: storage.NetworkEntityInfo_builder{
					Type: storage.NetworkEntityInfo_DEPLOYMENT,
					Id:   entityID,
					Deployment: storage.NetworkEntityInfo_Deployment_builder{
						Name: deploymentName,
					}.Build(),
				}.Build(),
				Scope: storage.NetworkEntity_Scope_builder{ClusterId: entityClusterID}.Build(),
			}.Build(),
			Properties: []*storage.NetworkBaselineConnectionProperties{
				storage.NetworkBaselineConnectionProperties_builder{
					Ingress:  flowIsIngress,
					Port:     flowPort,
					Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
				}.Build(),
			},
		}.Build(),
	})

	return baseline
}

func GetBaselineWithInternet(
	entityClusterID string,
	flowIsIngress bool,
	flowPort uint32,
) *storage.NetworkBaseline {
	baseline := fixtures.GetNetworkBaseline()
	baseline.SetPeers([]*storage.NetworkBaselinePeer{
		storage.NetworkBaselinePeer_builder{
			Entity: storage.NetworkEntity_builder{
				Info: storage.NetworkEntityInfo_builder{
					Type: storage.NetworkEntityInfo_INTERNET,
					Id:   networkgraph.InternetExternalSourceID,
				}.Build(),
				Scope: storage.NetworkEntity_Scope_builder{ClusterId: entityClusterID}.Build(),
			}.Build(),
			Properties: []*storage.NetworkBaselineConnectionProperties{
				storage.NetworkBaselineConnectionProperties_builder{
					Ingress:  flowIsIngress,
					Port:     flowPort,
					Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
				}.Build(),
			},
		}.Build(),
	})

	return baseline
}
