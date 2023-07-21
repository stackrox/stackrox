package metrics

import (
	"context"

	"github.com/stackrox/rox/pkg/sync"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

// GRPCMetrics provides an grpc interceptor which monitors API calls and recovers from panics and provides a method to get the metrics
//
//go:generate mockgen-wrapper
type GRPCMetrics interface {
	UnaryMonitoringInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error)
	GetMetrics() (map[string]map[codes.Code]int64, map[string]map[string]int64)
}

var (
	grpcInit    sync.Once
	grpcMetrics GRPCMetrics
)

// NewGRPCMetrics returns a new GRPCMetrics object
func NewGRPCMetrics() GRPCMetrics {
	return &grpcMetricsImpl{
		allMetrics: make(map[string]*perPathGRPCMetrics),
	}
}

// GRPCSingleton returns a singleton of GRPCMetrics.
func GRPCSingleton() GRPCMetrics {
	grpcInit.Do(func() {
		grpcMetrics = NewGRPCMetrics()
	})
	return grpcMetrics
}
