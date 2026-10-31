package service

import (
	"context"

	"github.com/stackrox/rox/central/operatorremediation/datasource"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/registries"
	"github.com/stackrox/rox/pkg/sync"
	"k8s.io/client-go/dynamic"
)

// Service is the gRPC service that generates OpenShift cluster-upgrade remediation reports.
type Service interface {
	grpc.APIService

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)

	v1.ClusterRemediationServiceServer
}

func newService(images *datasource.CentralImages, dyn dynamic.Interface, regSet registries.Set) Service {
	return &serviceImpl{
		images: images,
		dyn:    dyn,
		regSet: regSet,
		jobs:   make(map[string]*jobState),
		mu:     &sync.Mutex{},
	}
}
