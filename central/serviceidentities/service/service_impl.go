package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/serviceidentities/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/sac/resources"
	"google.golang.org/grpc"
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.Administration)): {
			"/v1.ServiceIdentityService/GetServiceIdentities",
			"/v1.ServiceIdentityService/GetAuthorities",
		},
		user.With(permissions.Modify(resources.Administration)): {
			"/v1.ServiceIdentityService/CreateServiceIdentity",
		},
	})
)

// IdentityService is the struct that manages the Service Identity API
type serviceImpl struct {
	v1.UnimplementedServiceIdentityServiceServer

	dataStore datastore.DataStore
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterServiceIdentityServiceServer(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterServiceIdentityServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

// GetServiceIdentities returns the currently defined service identities.
func (s *serviceImpl) GetServiceIdentities(ctx context.Context, _ *v1.Empty) (*v1.ServiceIdentityResponse, error) {
	serviceIdentities, err := s.dataStore.GetServiceIdentities(ctx)
	if err != nil {
		return nil, err
	}
	return &v1.ServiceIdentityResponse{
		Identities: serviceIdentities,
	}, nil
}

// CreateServiceIdentity generates a new key and certificate for a service.
// The key and certificate are not retained and can not be retrieved except
// in the response to this API call.
func (s *serviceImpl) CreateServiceIdentity(ctx context.Context, request *v1.CreateServiceIdentityRequest) (*v1.CreateServiceIdentityResponse, error) {
	if request == nil {
		return nil, errors.Wrap(errox.InvalidArgs, "Request must be nonempty")
	}
	if request.GetId() == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "ID must be nonempty")
	}
	if request.GetType() == storage.ServiceType_UNKNOWN_SERVICE {
		return nil, errors.Wrap(errox.InvalidArgs, "Service type must be nonempty")
	}
	issuedCert, err := mtls.IssueNewCert(mtls.NewSubject(request.GetId(), request.GetType()))
	if err != nil {
		return nil, err
	}
	if err := s.dataStore.AddServiceIdentity(ctx, issuedCert.ID); err != nil {
		return nil, err
	}

	return &v1.CreateServiceIdentityResponse{
		Identity:       issuedCert.ID,
		CertificatePem: issuedCert.CertPEM,
		PrivateKeyPem:  issuedCert.KeyPEM,
	}, nil
}

// GetAuthorities returns the authorities currently in use.
func (s *serviceImpl) GetAuthorities(_ context.Context, _ *v1.Empty) (*v1.Authorities, error) {
	ca, err := mtls.CACertPEM()
	if err != nil {
		return nil, err
	}
	return &v1.Authorities{
		Authorities: []*v1.Authority{
			{
				CertificatePem: ca,
			},
		},
	}, nil
}
