package connection

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/rate"
	"github.com/stackrox/rox/pkg/utils"
)

// chainedRateLimiter evaluates each limiter in order. A limiter that does not
// accept the message type lets it pass; the first limiter that rejects wins.
type chainedRateLimiter struct {
	limiters []*rate.Limiter
}

func (c chainedRateLimiter) TryConsume(clientID string, msg *central.MsgFromSensor) (allowed bool, reason string) {
	for _, l := range c.limiters {
		if allowed, reason := l.TryConsume(clientID, msg); !allowed {
			return false, reason
		}
	}
	return true, ""
}

func (c chainedRateLimiter) OnClientDisconnect(clientID string) {
	for _, l := range c.limiters {
		l.OnClientDisconnect(clientID)
	}
}

func newSensorEventRateLimiters() chainedRateLimiter {
	return chainedRateLimiter{
		limiters: []*rate.Limiter{
			newVMIndexReportRateLimiter(),
			newNodeIndexReportRateLimiter(),
		},
	}
}

func newVMIndexReportRateLimiter() *rate.Limiter {
	return newWorkloadRateLimiter(
		"vm_index_reports",
		env.VMIndexReportRateLimit.FloatSetting(),
		env.VMIndexReportBucketCapacity.IntegerSetting(),
		func(msg *central.MsgFromSensor) bool {
			return msg.GetEvent().GetVirtualMachineIndexReport() != nil
		},
	)
}

func newNodeIndexReportRateLimiter() *rate.Limiter {
	return newWorkloadRateLimiter(
		"node_index_reports",
		env.NodeIndexReportRateLimit.FloatSetting(),
		env.NodeIndexReportBucketCapacity.IntegerSetting(),
		func(msg *central.MsgFromSensor) bool {
			return msg.GetEvent().GetIndexReport() != nil
		},
	)
}

func newWorkloadRateLimiter(
	name string,
	globalRate float64,
	bucketCapacity int,
	accepts func(*central.MsgFromSensor) bool,
) *rate.Limiter {
	rl, err := rate.NewLimiter(name, globalRate, bucketCapacity).ForWorkload(accepts)
	if err != nil {
		utils.Should(errors.Wrapf(err, "Creating rate-limiter for %s", name))
	}
	return rl
}
