// +build destructive

package tests

import (
	"context"
	"fmt"
	"testing"
	"time"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/retry"
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
	assert.NotZero(t, resp.NodeCount)
	assert.NotZero(t, resp.ViolationCount)
	assert.NotZero(t, resp.DeploymentCount)
	assert.NotZero(t, resp.ImageCount)
	assert.NotZero(t, resp.SecretCount)

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

	err = retry.WithRetry(func() error {
		resp := getSummaryCounts(t)
		// Don't use cluster count as the trigger because that deletion runs synchronously
		if resp.DeploymentCount == 0 {
			// All of these should be 0 with no cluster
			assert.Equal(t, 0, resp.ClusterCount)
			assert.Equal(t, 0, resp.DeploymentCount)
			assert.Equal(t, 0, resp.NodeCount)
			assert.Equal(t, 0, resp.SecretCount)
			assert.Equal(t, 0, resp.ViolationCount)
			return nil
		}
		return retry.MakeRetryable(fmt.Errorf("resp still has non zero values: %+v", resp))
	}, retry.Tries(10),
		retry.BetweenAttempts(func(_ int) {
			time.Sleep(time.Second)
		}),
		retry.OnFailedAttempts(func(err error) {
			log.Error(err.Error())
		}))
	require.NoError(t, err)
}
