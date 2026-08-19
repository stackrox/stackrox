package service

import (
	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/pkg/grpc"
)

type Service interface {
	grpc.APIService
	v2.AIWorkloadServiceServer
}
