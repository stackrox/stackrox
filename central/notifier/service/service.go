package service

import (
	"context"

	"github.com/stackrox/rox/central/notifier/datastore"
	"github.com/stackrox/rox/central/notifier/policycleaner"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/cryptoutils"
	"github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/integrationhealth"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/notifier"
)

var (
	log = logging.LoggerForModule()
)

// Service provides the interface to the microservice that serves alert data.
type Service interface {
	grpc.APIService

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)

	v1.NotifierServiceServer
}

// New returns a new Service instance using the given DataStore.
func New(storage datastore.DataStore,
	processor notifier.Processor,
	policyCleaner policycleaner.PolicyCleaner,
	reporter integrationhealth.Reporter,
	cryptoCodec cryptoutils.CryptoCodec,
	cryptoKey string) Service {
	return &serviceImpl{
		storage:       storage,
		processor:     processor,
		policyCleaner: policyCleaner,
		reporter:      reporter,
		cryptoCodec:   cryptoCodec,
		cryptoKey:     cryptoKey,
	}
}
