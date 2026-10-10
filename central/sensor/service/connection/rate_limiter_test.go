package connection

import (
	"testing"

	"github.com/stackrox/rox/generated/internalapi/central"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	virtualmachineV1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/rate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewNodeIndexReportRateLimiter_Defaults(t *testing.T) {
	t.Setenv(env.NodeIndexReportRateLimit.EnvVar(), "")
	t.Setenv(env.NodeIndexReportBucketCapacity.EnvVar(), "")

	rl := newNodeIndexReportRateLimiter()
	require.NotNil(t, rl)
	assert.Equal(t, "node_index_reports", rl.WorkloadName())
	assert.Equal(t, 0.2, rl.GlobalRate())
	assert.Equal(t, 50, rl.BucketCapacity())
}

func TestNewNodeIndexReportRateLimiter_NegativeRateFallsBackToDefault(t *testing.T) {
	t.Setenv(env.NodeIndexReportRateLimit.EnvVar(), "-1")

	rl := newNodeIndexReportRateLimiter()
	require.NotNil(t, rl)
	assert.Equal(t, 0.2, rl.GlobalRate())
}

func TestNewNodeIndexReportRateLimiter_IgnoresNonIndexReports(t *testing.T) {
	t.Setenv(env.NodeIndexReportRateLimit.EnvVar(), "1")
	t.Setenv(env.NodeIndexReportBucketCapacity.EnvVar(), "1")

	rl := newNodeIndexReportRateLimiter()
	require.NotNil(t, rl)

	allowed, reason := rl.TryConsume("cluster-1", nodeIndexMsg("node-1"))
	require.True(t, allowed)
	assert.Empty(t, reason)

	allowed, reason = rl.TryConsume("cluster-1", nodeIndexMsg("node-2"))
	assert.False(t, allowed)
	assert.Equal(t, rate.ReasonRateLimitExceeded, reason)

	allowed, reason = rl.TryConsume("cluster-1", nodeInventoryMsg("node-3"))
	assert.True(t, allowed, "node inventory must not be rate limited")
	assert.Empty(t, reason)

	allowed, reason = rl.TryConsume("cluster-1", vmIndexMsg("vm-1"))
	assert.True(t, allowed, "VM index reports must not be consumed by the node limiter")
	assert.Empty(t, reason)
}

func TestNewVMIndexReportRateLimiter_IgnoresNodeIndexReports(t *testing.T) {
	t.Setenv(env.VMIndexReportRateLimit.EnvVar(), "1")
	t.Setenv(env.VMIndexReportBucketCapacity.EnvVar(), "1")

	rl := newVMIndexReportRateLimiter()
	require.NotNil(t, rl)

	allowed, _ := rl.TryConsume("cluster-1", vmIndexMsg("vm-1"))
	require.True(t, allowed)

	allowed, reason := rl.TryConsume("cluster-1", vmIndexMsg("vm-2"))
	assert.False(t, allowed)
	assert.Equal(t, rate.ReasonRateLimitExceeded, reason)

	allowed, reason = rl.TryConsume("cluster-1", nodeIndexMsg("node-1"))
	assert.True(t, allowed, "node index reports must not be consumed by the VM limiter")
	assert.Empty(t, reason)
}

func TestChainedRateLimiter_IsolatesWorkloads(t *testing.T) {
	t.Setenv(env.VMIndexReportRateLimit.EnvVar(), "1")
	t.Setenv(env.VMIndexReportBucketCapacity.EnvVar(), "1")
	t.Setenv(env.NodeIndexReportRateLimit.EnvVar(), "1")
	t.Setenv(env.NodeIndexReportBucketCapacity.EnvVar(), "1")

	rl := newSensorEventRateLimiters()

	allowed, _ := rl.TryConsume("cluster-1", vmIndexMsg("vm-1"))
	require.True(t, allowed)
	allowed, reason := rl.TryConsume("cluster-1", vmIndexMsg("vm-2"))
	assert.False(t, allowed)
	assert.Equal(t, rate.ReasonRateLimitExceeded, reason)

	allowed, _ = rl.TryConsume("cluster-1", nodeIndexMsg("node-1"))
	require.True(t, allowed, "exhausted VM limiter must not reject node index reports")
	allowed, reason = rl.TryConsume("cluster-1", nodeIndexMsg("node-2"))
	assert.False(t, allowed)
	assert.Equal(t, rate.ReasonRateLimitExceeded, reason)

	allowed, _ = rl.TryConsume("cluster-1", nodeInventoryMsg("node-3"))
	assert.True(t, allowed, "node inventory must pass both limiters")
}

func nodeIndexMsg(id string) *central.MsgFromSensor {
	return &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_Event{
			Event: &central.SensorEvent{
				Id: id,
				Resource: &central.SensorEvent_IndexReport{
					IndexReport: &v4.IndexReport{HashId: id},
				},
			},
		},
	}
}

func nodeInventoryMsg(id string) *central.MsgFromSensor {
	return &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_Event{
			Event: &central.SensorEvent{
				Id: id,
				Resource: &central.SensorEvent_NodeInventory{
					NodeInventory: &storage.NodeInventory{NodeId: id, NodeName: id},
				},
			},
		},
	}
}

func vmIndexMsg(id string) *central.MsgFromSensor {
	return &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_Event{
			Event: &central.SensorEvent{
				Id: id,
				Resource: &central.SensorEvent_VirtualMachineIndexReport{
					VirtualMachineIndexReport: &virtualmachineV1.IndexReportEvent{
						Id: id,
						Index: &virtualmachineV1.IndexReport{
							VsockCid: "100",
						},
					},
				},
			},
		},
	}
}
