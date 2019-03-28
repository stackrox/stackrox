package service

import (
	"context"
	"crypto/x509"
	"net/http"
	"strings"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/grpc/authz/allow"
	"github.com/stackrox/rox/pkg/logging"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	log = logging.LoggerForModule()

	x509Err = x509.UnknownAuthorityError{}.Error()
)

type serviceImpl struct {
	client http.Client
}

func (s *serviceImpl) URLHasValidCert(ctx context.Context, req *central.URLHasValidCertRequest) (*central.URLHasValidCertResponse, error) {
	if !strings.HasPrefix(req.GetUrl(), "https://") {
		return nil, status.Errorf(codes.InvalidArgument, "url %q must start with https", req.GetUrl())
	}
	_, err := s.client.Get(req.GetUrl())
	if err == nil {
		return &central.URLHasValidCertResponse{
			Result: central.URLHasValidCertResponse_REQUEST_SUCCEEDED,
		}, nil
	}
	errStr := err.Error()
	if strings.Contains(errStr, x509Err) {
		return &central.URLHasValidCertResponse{
			Result:  central.URLHasValidCertResponse_CERT_SIGNED_BY_UNKNOWN_AUTHORITY,
			Details: errStr,
		}, nil
	}
	if strings.Contains(errStr, "x509:") {
		return &central.URLHasValidCertResponse{
			Result:  central.URLHasValidCertResponse_CERT_SIGNING_AUTHORITY_VALID_BUT_OTHER_ERROR,
			Details: errStr,
		}, nil
	}
	return &central.URLHasValidCertResponse{
		Result:  central.URLHasValidCertResponse_OTHER_GET_ERROR,
		Details: errStr,
	}, nil
}

// New creates a new Service.
func New() Service {
	return &serviceImpl{
		client: http.Client{
			Timeout: 20 * time.Second,
		},
	}
}

func (s *serviceImpl) RegisterServiceServer(server *grpc.Server) {
	central.RegisterDevelopmentServiceServer(server, s)
}

func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return nil
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, allow.Anonymous().Authorized(ctx, fullMethodName)
}
