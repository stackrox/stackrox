package tests

import (
	"context"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSummaryData(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	conn, err := grpcConnection()
	require.NoError(t, err)

	service := v1.NewSummaryServiceClient(conn)
	countsResp, err := service.GetSummaryCounts(ctx, &v1.Empty{})
	require.NoError(t, err)

	assert.True(t, countsResp.GetNumClusters() >= 1)
	assert.True(t, countsResp.GetNumDeployments() >= 2)
	assert.True(t, countsResp.GetNumNodes() >= 1)
}
