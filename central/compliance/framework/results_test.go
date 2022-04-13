package framework

import (
	"errors"
	"testing"

	"github.com/stackrox/stackrox/pkg/concurrency"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithResults(t *testing.T) {
	t.Parallel()

	results := newResults()
	stopSig := concurrency.NewErrorSignal()
	ctx := newToplevelContext("", testDomain, nil, results, &stopSig)

	var deploymentCheck = func(ctx ComplianceContext) {
		PassNow(ctx, ctx.Target().Deployment().GetId())
		Fail(ctx, "should not happen")
	}

	testErr := errors.New("only running for first deployment")
	doRun(ctx, func(ctx ComplianceContext) {
		for _, deployment := range ctx.Domain().Deployments() {
			RunForTarget(ctx, deployment, deploymentCheck)
			Abort(ctx, testErr)
		}
	})

	assert.Equal(t, testErr, results.Error())
	deployment1Results := results.ForChild(testDomain.Deployments()[0])
	require.NotNil(t, deployment1Results)
	assert.NoError(t, deployment1Results.Error())
	require.Len(t, deployment1Results.Evidence(), 1)
	assert.Equal(t, deployment1Results.Evidence()[0], EvidenceRecord{
		Status:  PassStatus,
		Message: testDeployments[0].GetId(),
	})

	deployment2Results := results.ForChild(testDomain.Deployments()[1])
	assert.Nil(t, deployment2Results)
}
