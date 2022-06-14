package framework

import (
	"errors"
	"testing"

	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/concurrency"
	"github.com/stackrox/stackrox/pkg/set"
	"github.com/stretchr/testify/assert"
)

func TestContextAccessChecksForError(t *testing.T) {
	syncSig := concurrency.NewSignal()

	seenNodeIDs := set.NewStringSet()

	var checkFn = func(ctx ComplianceContext, node *storage.Node) {
		syncSig.Wait()
		_ = ctx.Target()
		seenNodeIDs.Add(node.GetId())
	}

	stopSig := concurrency.NewErrorSignal()
	ctx := newToplevelContext("", testDomain, nil, newResults(), &stopSig)
	go func() {
		stopSig.SignalWithError(errors.New("error"))
		syncSig.Signal()
	}()

	assert.Panics(t, func() { ForEachNode(ctx, checkFn) })
	assert.Empty(t, seenNodeIDs)
}

func TestContextStopAbortsCurrentCheckOnly(t *testing.T) {
	seenNodeIDs := set.NewStringSet()
	expectedNodeIDs := set.NewStringSet(testNodes[1].GetId())

	var checkFn = func(ctx ComplianceContext, node *storage.Node) {
		if node.GetId() == testNodes[0].GetId() {
			Abort(ctx, nil)
		}
		seenNodeIDs.Add(node.GetId())
	}

	stopSig := concurrency.NewErrorSignal()
	ctx := newToplevelContext("", testDomain, nil, newResults(), &stopSig)

	ForEachNode(ctx, checkFn)
	assert.Equal(t, expectedNodeIDs, seenNodeIDs)
}
