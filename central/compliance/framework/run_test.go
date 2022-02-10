package framework

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/compliance/framework"
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

	testPods = []*storage.Pod{
		{
			Id: uuid.NewV4().String(),
		},
		{
			Id: uuid.NewV4().String(),
		},
	}

	testMachineConfigs = map[string][]string{
		"standard": {
			"config1",
			"config2",
		},
	}

	testDomain = newComplianceDomain(testCluster, testNodes, testDeployments, testPods, testMachineConfigs)
)

func TestEmptyRun(t *testing.T) {
	t.Parallel()

	run, err := newComplianceRun()
	require.NoError(t, err)

	err = run.Run(context.Background(), "standard", testDomain, nil)
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

	nodeCheck := NewCheckFromFunc(
		CheckMetadata{ID: "node-check", Scope: framework.NodeKind},
		func(ctx ComplianceContext) {
			ForEachNode(ctx, nodeCheckFn)
		})
	deploymentCheck := NewCheckFromFunc(
		CheckMetadata{ID: "deployment-check", Scope: framework.DeploymentKind},
		func(ctx ComplianceContext) {
			ForEachDeployment(ctx, deploymentCheckFn)
		})

	clusterCheck := NewCheckFromFunc(
		CheckMetadata{ID: "cluster-check", Scope: framework.ClusterKind},
		clusterCheckFn)

	run, err := newComplianceRun(clusterCheck, nodeCheck, deploymentCheck)
	require.NoError(t, err)

	err = run.Run(context.Background(), "standard", testDomain, nil)
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
	clusterCheck := NewCheckFromFunc(
		CheckMetadata{ID: "cluster-check", Scope: framework.ClusterKind},
		clusterCheckFn)

	run, err := newComplianceRun(clusterCheck)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		assert.Error(t, run.Run(ctx, "standard", testDomain, nil))
	}()

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
	clusterCheck := NewCheckFromFunc(
		CheckMetadata{ID: "cluster-check", Scope: framework.ClusterKind},
		clusterCheckFn)

	run, err := newComplianceRun(clusterCheck)
	require.NoError(t, err)

	go func() {
		assert.Error(t, run.Run(context.Background(), "standard", testDomain, nil))
	}()

	run.Terminate(errors.New("terminating run"))
	syncSig.Signal()

	err = run.Wait()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "terminating run")
}
