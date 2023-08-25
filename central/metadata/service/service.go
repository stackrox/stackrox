package service

import (
	"context"

	"github.com/stackrox/rox/central/globaldb"
	systemInfoStorage "github.com/stackrox/rox/central/systeminfo/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// Service is the GRPC service interface that provides the entry point for processing deployment events.
type Service interface {
	grpc.APIService
	v1.MetadataServiceServer

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)
}

// New returns a new instance of service.
func New() Service {
	return &serviceImpl{
		db:              globaldb.GetPostgres(),
		systemInfoStore: systemInfoStorage.Singleton(),
	}
}
