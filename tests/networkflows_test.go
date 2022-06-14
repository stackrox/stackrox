package tests

import (
	"context"
	"testing"
	"time"

	"github.com/gogo/protobuf/types"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/testutils/centralgrpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func hasEdges(graph *v1.NetworkGraph) bool {
	for _, node := range graph.GetNodes() {
		if len(node.GetOutEdges()) > 0 {
			return true
		}
	}
	return false
}

func TestStackroxNetworkFlows(t *testing.T) {
	t.Parallel()

	conn := centralgrpc.GRPCConnectionToCentral(t)

	clustersService := v1.NewClustersServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	clusters, err := clustersService.GetClusters(ctx, &v1.GetClustersRequest{})
	cancel()

	require.NoError(t, err)
	var mainCluster *storage.Cluster
	for _, cluster := range clusters.GetClusters() {
		if cluster.GetName() == "remote" {
			mainCluster = cluster
			break
		}
	}
	require.NotNil(t, mainCluster, "cluster with name remote not found")

	clusterID := mainCluster.GetId()

	service := v1.NewNetworkGraphServiceClient(conn)

	var graph *v1.NetworkGraph
	timeout := time.NewTimer(5 * time.Minute)
	ticker := time.NewTicker(5 * time.Second)
	for {
		select {
		case <-timeout.C:
			t.Fatal("Failed to get the correct edges in 5 minutes")
		case <-ticker.C:
			ctx, cancel = context.WithTimeout(context.Background(), time.Minute)
			graph, err = service.GetNetworkGraph(ctx, &v1.NetworkGraphRequest{
				ClusterId: clusterID,
				Query:     "namespace:stackrox",
				Since:     types.TimestampNow(),
			})
			cancel()
			if err != nil {
				log.Errorf("error getting graph: %v", err)
			}
		}
		if hasEdges(graph) {
			break
		}
	}

	require.NoError(t, err)

	type deploymentConn struct {
		srcName, targetName string
	}

	var conns []deploymentConn

	internetIngressDeployments := set.NewStringSet()

	for _, node := range graph.GetNodes() {
		if node.GetEntity().GetType() != storage.NetworkEntityInfo_INTERNET && node.GetEntity().GetDeployment().GetNamespace() != "stackrox" {
			continue
		}

		for otherNodeIdx := range node.GetOutEdges() {
			otherNode := graph.GetNodes()[otherNodeIdx]
			if otherNode.GetEntity().GetDeployment().GetNamespace() != "stackrox" {
				continue
			}

			if node.GetEntity().GetType() == storage.NetworkEntityInfo_INTERNET {
				internetIngressDeployments.Add(otherNode.GetEntity().GetDeployment().GetName())
			} else {
				conns = append(conns, deploymentConn{
					srcName:    node.GetEntity().GetDeployment().GetName(),
					targetName: otherNode.GetEntity().GetDeployment().GetName()})
			}
		}
	}

	expectedConns := []deploymentConn{
		{srcName: "collector", targetName: "sensor"},
		{srcName: "sensor", targetName: "central"},
	}

	assert.Subset(t, conns, expectedConns, "expected connections not found")
	assert.NotContains(t, internetIngressDeployments.AsSlice(), "collector", "collector should not have internet ingress")
	// Readiness/health probes might show up as internet ingress, so disable this for now.
	// TODO(ROX-2034): Re-enable.
	// assert.NotContains(t, internetIngressDeployments.AsSlice(), "sensor", "sensor should not have internet ingress")
}
