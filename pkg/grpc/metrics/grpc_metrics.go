package metrics

import (
	"context"

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

// NewGRPCMetrics returns a new GRPCMetrics object
func NewGRPCMetrics() GRPCMetrics {
	return &grpcMetricsImpl{
		allMetrics: make(map[string]*perPathGRPCMetrics),
	}
}
