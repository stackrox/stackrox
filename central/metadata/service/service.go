package service

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/central/globaldb"
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
	return &serviceImpl{db: globaldb.GetPostgres()}
}

// NewForPostgresTestOnly returns a new instance of service with a test Postgres connection.
func NewForPostgresTestOnly(_ *testing.T, pool *pgxpool.Pool) Service {
	return &serviceImpl{db: pool}
}
