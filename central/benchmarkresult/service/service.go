package service

import (
	"context"

	"github.com/hashicorp/golang-lru"
	benchmarkscanStore "github.com/stackrox/rox/central/benchmarkscan/store"
	benchmarkscheduleStore "github.com/stackrox/rox/central/benchmarkschedule/store"
	notifierProcessor "github.com/stackrox/rox/central/notifier/processor"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/logging"
)

const cacheSize = 100

var (
	log = logging.LoggerForModule()
)

// Service provides the interface to the microservice that serves alert data.
type Service interface {
	grpc.APIService

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)

	PostBenchmarkResult(ctx context.Context, request *v1.BenchmarkResult) (*v1.Empty, error)
}

// New returns a new instance of Service using the input storage and processing mechanisms.
func New(resultStore benchmarkscanStore.Store, scheduleStore benchmarkscheduleStore.Store, notificationsProcessor notifierProcessor.Processor) Service {
	cache, err := lru.New(cacheSize)
	if err != nil {
		// This only happens in extreme cases (at this time, for invalid size only).
		panic(err)
	}
	return &serviceImpl{
		resultStore:   resultStore,
		scheduleStore: scheduleStore,
		cache:         cache,
		notificationsProcessor: notificationsProcessor,
	}
}
