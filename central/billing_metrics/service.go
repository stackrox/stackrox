package billingmetrics

import (
	"context"

	// "github.com/stackrox/rox/central/billing_metrics/backend"
	"github.com/stackrox/rox/central/apitoken/backend"
	pkgGRPC "github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// Service provides the interface to the svc that handles API keys.
type Service interface {
	pkgGRPC.APIService

	GetMaximum(ctx context.Context, metric string) (context.Context, error)
}

// New returns a ready-to-use instance of Service.
func New(backend backend.Backend, roles roleDS.DataStore) Service {
	return &serviceImpl{backend: backend, roles: roles}
}
