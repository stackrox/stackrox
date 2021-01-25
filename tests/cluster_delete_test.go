// +build destructive

package tests

import (
	"context"
	"testing"
	"time"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type summaryCountsResp struct {
	ClusterCount, NodeCount, ViolationCount, DeploymentCount, ImageCount, SecretCount int
}

func getSummaryCounts(t *testing.T) summaryCountsResp {
	var resp summaryCountsResp
	makeGraphQLRequest(t, `
		query summary_counts {
			clusterCount
			nodeCount
			violationCount
			deploymentCount
			imageCount
			secretCount
		}
	`, map[string]interface{}{}, &resp, timeout)
	return resp
}

func TestClusterDeletion(t *testing.T) {
	resp := getSummaryCounts(t)
	assert.NotZero(t, resp.ClusterCount)
	// ROX-6391: NodeCount starts at zero
	// assert.NotZero(t, resp.NodeCount)
	assert.NotZero(t, resp.ViolationCount)
	assert.NotZero(t, resp.DeploymentCount)
	assert.NotZero(t, resp.ImageCount)
	assert.NotZero(t, resp.SecretCount)
	log.Infof("the initial counts are: %+v", resp)

	conn := testutils.GRPCConnectionToCentral(t)
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
	for {
		previous := resp

		time.Sleep(5 * time.Second)

		resp := getSummaryCounts(t)
		if resp.DeploymentCount == 0 {
			if resp.ClusterCount == 0 &&
				resp.NodeCount == 0 &&
				resp.SecretCount == 0 &&
				resp.ViolationCount == 0 {
				log.Infof("objects have all drained to 0")
				return
			}
			log.Infof("resp still has non zero values: %+v", resp)
		} else {
			log.Infof("deployment count is still not zero: %d", resp.DeploymentCount)
		}

		if previous.DeploymentCount > 0 && previous.DeploymentCount > resp.DeploymentCount {
			noChangeCount = 0
		} else {
			noChangeCount++
		}
		loopCount++

		require.LessOrEqual(t, noChangeCount, 20)
		require.LessOrEqual(t, loopCount, 240)
	}
}
