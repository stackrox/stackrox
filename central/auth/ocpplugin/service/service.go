package service

import (
	"context"

	"github.com/stackrox/rox/central/jwt"
	"github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	once sync.Once
	s    Service
)

// Service provides the interface to the microservice that serves tokens for the OCP plugin.
type Service interface {
	grpc.APIService

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)
}

// Singleton returns a new auth service instance.
func Singleton() Service {
	once.Do(func() {
		source := newSource()
		issuer, err := jwt.IssuerFactorySingleton().CreateIssuer(source)
		utils.Must(err)
		s = &serviceImpl{
			issuer: issuer,
		}
	})
	return s
}
