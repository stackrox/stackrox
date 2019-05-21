package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/stackrox/rox/central/clientca/manager"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/dberrors"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.ClientTrustCerts)): {
			"/v1.CertificateService/GetClientCertificateAuthorities",
			"/v1.CertificateService/GetClientCertificateAuthority",
		},
		user.With(permissions.Modify(resources.ClientTrustCerts)): {
			"/v1.CertificateService/CreateClientCertificateAuthority",
			"/v1.CertificateService/DeleteClientCertificateAuthority",
		},
	})
)

type service struct {
	clientCAManager manager.ClientCAManager
}

func newService(manager manager.ClientCAManager) *service {
	return &service{clientCAManager: manager}
}

func (s *service) RegisterServiceServer(server *grpc.Server) {
	v1.RegisterCertificateServiceServer(server, s)
}

func (s *service) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterCertificateServiceHandler(ctx, mux, conn)
}

func (s *service) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

func (s *service) GetClientCertificateAuthorities(ctx context.Context, _ *v1.Empty) (*v1.GetClientCertificateAuthoritiesResponse, error) {
	certs := s.clientCAManager.GetAllClientCAs(ctx)
	return &v1.GetClientCertificateAuthoritiesResponse{
		Certificates: certs,
	}, nil
}

func (s *service) GetClientCertificateAuthority(ctx context.Context, req *v1.ResourceByID) (*v1.GetClientCertificateAuthorityResponse, error) {
	cert, ok := s.clientCAManager.GetClientCA(ctx, req.GetId())
	if !ok {
		return nil, dberrors.ErrNotFound{Type: "ClientCA", ID: req.GetId()}
	}
	return &v1.GetClientCertificateAuthorityResponse{
		Certificate: cert,
	}, nil
}

func (s *service) CreateClientCertificateAuthority(ctx context.Context, req *v1.CreateClientCertificateAuthorityRequest) (*v1.CreateClientCertificateAuthorityResponse, error) {
	if req.GetPem() == "" {
		return nil, status.Error(codes.InvalidArgument, "No certificate PEM included")
	}
	cert, err := s.clientCAManager.AddClientCA(ctx, req.GetPem())
	if err != nil {
		return nil, err
	}
	return &v1.CreateClientCertificateAuthorityResponse{
		Certificate: cert,
	}, nil
}

func (s *service) DeleteClientCertificateAuthority(ctx context.Context, req *v1.ResourceByID) (*v1.Empty, error) {
	err := s.clientCAManager.RemoveClientCA(ctx, req.GetId())
	return &v1.Empty{}, err
}
