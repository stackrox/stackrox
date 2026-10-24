package service

import (
	"context"

	"github.com/stackrox/rox/central/operatorremediation/datasource"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc"
	"k8s.io/client-go/dynamic"
)

// Service is the gRPC service exposing operator image-remediation guidance.
type Service interface {
	grpc.APIService

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)

	v1.OperatorRemediationServiceServer
}

func newService(images *datasource.CentralImages, dyn dynamic.Interface) Service {
	return &serviceImpl{images: images, dyn: dyn}
}
