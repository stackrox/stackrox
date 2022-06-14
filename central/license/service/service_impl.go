package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"google.golang.org/grpc"
)

type service struct{}

func newService() *service {
	return &service{}
}

func (s *service) RegisterServiceServer(server *grpc.Server) {
	v1.RegisterLicenseServiceServer(server, s)
}

func (s *service) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterLicenseServiceHandler(ctx, mux, conn)
}

func (s *service) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, nil
}

func (s *service) GetActiveLicenseKey(context.Context, *v1.Empty) (*v1.GetActiveLicenseKeyResponse, error) {
	return &v1.GetActiveLicenseKeyResponse{}, nil
}

func (s *service) GetLicenses(ctx context.Context, req *v1.GetLicensesRequest) (*v1.GetLicensesResponse, error) {
	return &v1.GetLicensesResponse{}, nil
}

func (s *service) AddLicense(ctx context.Context, req *v1.AddLicenseRequest) (*v1.AddLicenseResponse, error) {
	return &v1.AddLicenseResponse{
		Accepted: true,
	}, nil
}

func (s *service) GetActiveLicenseExpiration(ctx context.Context, _ *v1.Empty) (*v1.GetActiveLicenseExpirationResponse, error) {
	return &v1.GetActiveLicenseExpirationResponse{}, nil
}
