package tests

import (
	"context"
	"testing"
	"time"

	"github.com/gogo/protobuf/types"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStackroxNetworkFlows(t *testing.T) {
	conn := testutils.GRPCConnectionToCentral(t)

	clustersService := v1.NewClustersServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	clusters, err := clustersService.GetClusters(ctx, &v1.Empty{})
	cancel()

	require.NoError(t, err)
	require.Len(t, clusters.GetClusters(), 1)

	clusterID := clusters.GetClusters()[0].GetId()

	service := v1.NewNetworkGraphServiceClient(conn)

	ctx, cancel = context.WithTimeout(context.Background(), time.Minute)
	graph, err := service.GetNetworkGraph(ctx, &v1.NetworkGraphRequest{
		ClusterId: clusterID,
		Query:     "namespace:stackrox",
		Since:     types.TimestampNow(),
	})
	cancel()

	require.NoError(t, err)

	type deploymentConn struct {
		srcName, targetName string
	}

	var conns []deploymentConn

	for _, node := range graph.GetNodes() {
		if node.GetEntity().GetDeployment().GetNamespace() != "stackrox" {
			continue
		}

		for otherNodeIdx := range node.GetOutEdges() {
			otherNode := graph.GetNodes()[otherNodeIdx]
			if otherNode.GetEntity().GetDeployment().GetNamespace() != "stackrox" {
				continue
			}

			conns = append(conns, deploymentConn{
				srcName:    node.GetEntity().GetDeployment().GetName(),
				targetName: otherNode.GetEntity().GetDeployment().GetName()})
		}
	}

	expectedConns := []deploymentConn{
		{srcName: "collector", targetName: "sensor"},
		{srcName: "sensor", targetName: "central"},
	}

	assert.Subset(t, conns, expectedConns, "expected connections not found")
}
