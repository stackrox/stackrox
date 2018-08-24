package service

import (
	"context"

	"github.com/stackrox/rox/central/apitoken/cachedstore"
	"github.com/stackrox/rox/central/apitoken/parser"
	"github.com/stackrox/rox/central/apitoken/signer"
	rolestore "github.com/stackrox/rox/central/role/store"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc"
)

// Service provides the interface to the svc that handles API keys.
type Service interface {
	v1.APITokenServiceServer

	grpc.APIService

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)
}

// New returns a ready-to-use instance of Service.
func New(signer signer.Signer, parser parser.Parser, roleStore rolestore.Store, tokenStore cachedstore.CachedStore) Service {
	return &serviceImpl{signer: signer, parser: parser, roleStore: roleStore, tokenStore: tokenStore}
}
