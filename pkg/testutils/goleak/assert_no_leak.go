package goleak

import (
	"testing"

	"go.uber.org/goleak"
)

func AssertNoGoroutineLeaks(t testing.TB) {
	t.Cleanup(func() {
		goleak.VerifyNone(t,
			// Ignore a known leak: https://github.com/DataDog/dd-trace-go/issues/1469
			goleak.IgnoreTopFunction("github.com/golang/glog.(*fileSink).flushDaemon"),
			// Ignore a known leak caused by importing the GCP cscc SDK.
			goleak.IgnoreTopFunction("go.opencensus.io/stats/view.(*worker).start"),
			// Ignore a known leak from https://github.com/hashicorp/golang-lru/blob/v2.0.7/expirable/expirable_lru.go#L77-L80
			goleak.IgnoreTopFunction("github.com/hashicorp/golang-lru/v2/expirable.NewLRU[...].func1"),
		)
	})
}
