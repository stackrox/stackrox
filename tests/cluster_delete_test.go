//go:build destructive

package tests

import (
	"context"
	"testing"
	"time"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/testutils/centralgrpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type allCounts struct {
	summaryCountsResp
	PodCount int
}

func allIntsZero(ints ...int) bool {
	for _, i := range ints {
		if i != 0 {
			return false
		}
	}
	return true
}

func (a *allCounts) AllZero() bool {
	return allIntsZero(a.PodCount, a.ClusterCount, a.NodeCount, a.ViolationCount, a.DeploymentCount, a.SecretCount)
}

type summaryCountsResp struct {
	ClusterCount, NodeCount, ViolationCount, DeploymentCount, SecretCount int
}

func getSummaryCounts(t *testing.T) summaryCountsResp {
	var resp summaryCountsResp
	makeGraphQLRequest(t, `
		query summary_counts {
			clusterCount
			nodeCount
			violationCount
			deploymentCount
			secretCount
		}
	`, map[string]interface{}{}, &resp, timeout)
	return resp
}

func getAllCounts(t *testing.T) allCounts {
	summaryCounts := getSummaryCounts(t)
	conn := centralgrpc.GRPCConnectionToCentral(t)
	podSvc := v1.NewPodServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	podsResp, err := podSvc.GetPods(ctx, &v1.RawQuery{})
	require.NoError(t, err)
	return allCounts{
		summaryCountsResp: summaryCounts,
		PodCount:          len(podsResp.Pods),
	}
}

func TestClusterDeletion(t *testing.T) {
	counts := getAllCounts(t)
	assert.NotZero(t, counts.ClusterCount)
	// ROX-6391: NodeCount starts at zero
	// assert.NotZero(t, resp.NodeCount)
	assert.NotZero(t, counts.ViolationCount)
	assert.NotZero(t, counts.DeploymentCount)
	assert.NotZero(t, counts.SecretCount)
	assert.NotZero(t, counts.PodCount)
	log.Infof("the initial counts are: %+v", counts)

	conn := centralgrpc.GRPCConnectionToCentral(t)
	service := v1.NewClustersServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	getClustersResp, err := service.GetClusters(ctx, &v1.GetClustersRequest{})
	require.NoError(t, err)
	cancel()

	for _, cluster := range getClustersResp.GetClusters() {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		_, err := service.DeleteCluster(ctx, &v1.ResourceByID{Id: cluster.GetId()})
		assert.NoError(t, err)
		cancel()
	}

	var noChangeCount, loopCount int
	previous := counts
	for {
		time.Sleep(5 * time.Second)

		counts := getAllCounts(t)
		if counts.DeploymentCount == 0 {
			if counts.AllZero() {
				log.Infof("objects have all drained to 0")
				return
			}
			log.Infof("resp still has non zero values: %+v", counts)
		} else {
			log.Infof("deployment count is still not zero: %d", counts.DeploymentCount)
		}

		if previous.DeploymentCount > 0 && previous.DeploymentCount > counts.DeploymentCount {
			noChangeCount = 0
		} else {
			noChangeCount++
		}
		loopCount++

		require.LessOrEqual(t, noChangeCount, 20)
		require.LessOrEqual(t, loopCount, 240)
		previous = counts
	}
}
