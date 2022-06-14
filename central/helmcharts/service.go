package helmcharts

import "github.com/stackrox/stackrox/pkg/grpc"

// NewService creates and returns a new service for downloading helm charts.
func NewService() grpc.APIServiceWithCustomRoutes {
	return &service{}
}
