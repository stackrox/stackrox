package metrics

import (
	"github.com/stackrox/rox/pkg/grpc/metrics"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	grpcInit    sync.Once
	grpcMetrics metrics.GRPCMetrics

	httpInit    sync.Once
	httpMetrics metrics.HTTPMetrics
)

// GRPCSingleton returns a singleton of a GRPCMetrics stuct
func GRPCSingleton() metrics.GRPCMetrics {
	grpcInit.Do(func() {
		grpcMetrics = metrics.NewGRPCMetrics()
	})
	return grpcMetrics
}

// HTTPSingleton returns a singleton of a HTTPMetrics stuct
func HTTPSingleton() metrics.HTTPMetrics {
	httpInit.Do(func() {
		httpMetrics = metrics.NewHTTPMetrics()
	})
	return httpMetrics
}
