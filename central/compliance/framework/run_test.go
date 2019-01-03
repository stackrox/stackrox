package framework

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	testCluster = &storage.Cluster{
		Id: uuid.NewV4().String(),
	}

	testDeployments = []*storage.Deployment{
		{
			Id: uuid.NewV4().String(),
		},
		{
			Id: uuid.NewV4().String(),
		},
	}

	testNodes = []*storage.Node{
		{
			Id: uuid.NewV4().String(),
		},
		{
			Id: uuid.NewV4().String(),
		},
	}

	testDomain = newComplianceDomain(testCluster, testNodes, testDeployments)
)

func TestEmptyRun(t *testing.T) {
	t.Parallel()

	run, err := newComplianceRun()
	require.NoError(t, err)

	err = run.Run(context.Background(), testDomain, nil)
	assert.NoError(t, err)
}

func TestOKRun(t *testing.T) {
	seenNodeIDs := set.NewStringSet()
	nodeCheckFn := func(ctx ComplianceContext, node *storage.Node) {
		seenNodeIDs.Add(node.GetId())
	}
	expectedNodeIDs := set.NewStringSet(testNodes[0].Id, testNodes[1].Id)

	seenDeploymentIDs := set.NewStringSet()
	deploymentCheckFn := func(ctx ComplianceContext, deployment *storage.Deployment) {
		seenDeploymentIDs.Add(deployment.GetId())
	}
	expectedDeploymentIDs := set.NewStringSet(testDeployments[0].Id, testDeployments[1].Id)

	seenClusterIDs := set.NewStringSet()
	clusterCheckFn := func(ctx ComplianceContext) {
		seenClusterIDs.Add(ctx.Domain().Cluster().Cluster().GetId())
	}
	expectedClusterIDs := set.NewStringSet(testCluster.Id)

	nodeCheck := NewCheckFromFunc("node-check", NodeKind, nil, func(ctx ComplianceContext) {
		ForEachNode(ctx, nodeCheckFn)
	})
	deploymentCheck := NewCheckFromFunc("deployment-check", DeploymentKind, nil, func(ctx ComplianceContext) {
		ForEachDeployment(ctx, deploymentCheckFn)
	})
	clusterCheck := NewCheckFromFunc("cluster-check", ClusterKind, nil, clusterCheckFn)

	run, err := newComplianceRun(clusterCheck, nodeCheck, deploymentCheck)
	require.NoError(t, err)

	err = run.Run(context.Background(), testDomain, nil)
	assert.NoError(t, err)

	assert.Equal(t, expectedNodeIDs, seenNodeIDs)
	assert.Equal(t, expectedDeploymentIDs, seenDeploymentIDs)
	assert.Equal(t, expectedClusterIDs, seenClusterIDs)
}

func TestRunWithContextError(t *testing.T) {
	syncSig := concurrency.NewSignal()
	clusterCheckFn := func(ctx ComplianceContext) {
		syncSig.Wait()
	}
	clusterCheck := NewCheckFromFunc("cluster-check", ClusterKind, nil, clusterCheckFn)

	run, err := newComplianceRun(clusterCheck)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	go run.Run(ctx, testDomain, nil)

	cancel()
	time.Sleep(100 * time.Millisecond)
	syncSig.Signal()

	err = run.Wait()
	require.Error(t, err)
	assert.Contains(t, err.Error(), context.Canceled.Error())
}

func TestRunWithTerminate(t *testing.T) {
	syncSig := concurrency.NewSignal()
	clusterCheckFn := func(ctx ComplianceContext) {
		syncSig.Wait()
	}
	clusterCheck := NewCheckFromFunc("cluster-check", ClusterKind, nil, clusterCheckFn)

	run, err := newComplianceRun(clusterCheck)
	require.NoError(t, err)

	go run.Run(context.Background(), testDomain, nil)

	run.Terminate(errors.New("terminating run"))
	syncSig.Signal()

	err = run.Wait()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "terminating run")
}
