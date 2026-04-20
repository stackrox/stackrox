package api_requests

import (
	"context"
	"time"

	"github.com/stackrox/rox/pkg/clientprofile"
	"github.com/stackrox/rox/pkg/eventual"
	"github.com/stackrox/rox/pkg/grpc/common/requestinterceptor"
	"github.com/stackrox/rox/pkg/logging"
)

var log = logging.LoggerForModule()

const handlerName = "api_requests"

var interceptor = eventual.New(
	eventual.WithType[*requestinterceptor.RequestInterceptor]().
		WithTimeout(5 * time.Minute).
		WithContextCallback(func(ctx context.Context) {
			log.Error("Request interceptor not provided within timeout; API request metrics will not be collected")
		}),
)

// SetInterceptor provides the RequestInterceptor.
func SetInterceptor(ri *requestinterceptor.RequestInterceptor) {
	interceptor.Set(ri)
}

// RegisterHandler adds or removes the RecordRequest handler on the
// RequestInterceptor based on whether any profile tracker is currently enabled.
// Blocks until the interceptor is provided via SetInterceptor, or the timeout
// expires. This is safe because runner.initialize runs in its own goroutine.
// Call this after reconfiguring trackers.
func RegisterHandler() {
	ri := interceptor.Get()
	if ri == nil {
		return
	}
	for _, t := range profileTrackers {
		if t.IsEnabled() {
			ri.Add(handlerName, recordRequest)
			log.Info("API request metrics handler registered")
			return
		}
	}
	ri.Remove(handlerName)
	log.Info("API request metrics handler removed (no enabled profile trackers)")
}

// recordRequest matches the request against all profiles and increments every
// matching tracker. If no profile matches, the unknown tracker is incremented.
func recordRequest(rp *requestinterceptor.RequestParams) {
	matched := false
	for name, ruleset := range builtinProfiles {
		if ruleset.CountMatched(rp, func(*clientprofile.Rule, clientprofile.Headers) {}) > 0 {
			if t, ok := profileTrackers[name]; ok {
				t.IncrementCounter(rp)
				matched = true
			}
		}
	}
	if !matched {
		if t, ok := profileTrackers[unknownProfile]; ok {
			t.IncrementCounter(rp)
		}
	}
}
