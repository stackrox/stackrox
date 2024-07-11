package framework

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stretchr/testify/assert"
)

func TestDoRunCatchesHalt(t *testing.T) {
	t.Parallel()

	var checkFn = func(ctx ComplianceContext) {
		halt(errors.New("some error"))
	}

	stopSig := concurrency.NewErrorSignal()
	ctx := newToplevelContext("", testDomain, nil, newResults(), &stopSig)
	assert.NotPanics(t, func() { doRun(ctx, checkFn) })
}

func TestForEachNode(t *testing.T) {
	t.Parallel()

	expectedNodeIDs := set.NewStringSet(testNodes[0].Id, testNodes[1].Id)

	seenNodeIDs := set.NewStringSet()
	var checkFn = func(ctx ComplianceContext, node *storage.Node) {
		seenNodeIDs.Add(node.GetId())
	}

	stopSig := concurrency.NewErrorSignal()
	ctx := newToplevelContext("", testDomain, nil, newResults(), &stopSig)
	ForEachNode(ctx, checkFn)
	assert.Equal(t, expectedNodeIDs, seenNodeIDs)
}

func TestForEachDeployment(t *testing.T) {
	t.Parallel()

	expectedDeploymentIDs := set.NewStringSet(testDeployments[0].Id, testDeployments[1].Id)

	seenDeploymentIDs := set.NewStringSet()
	var checkFn = func(ctx ComplianceContext, deployment *storage.Deployment) {
		seenDeploymentIDs.Add(deployment.GetId())
	}

	stopSig := concurrency.NewErrorSignal()
	ctx := newToplevelContext("", testDomain, nil, newResults(), &stopSig)
	ForEachDeployment(ctx, checkFn)
	assert.Equal(t, expectedDeploymentIDs, seenDeploymentIDs)
}
