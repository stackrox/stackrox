package service

import (
	"context"
	"crypto/x509"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	"github.com/stackrox/stackrox/central/sensor/service/connection"
	"github.com/stackrox/stackrox/generated/internalapi/central"
	"github.com/stackrox/stackrox/pkg/errox"
	"github.com/stackrox/stackrox/pkg/grpc/authz/allow"
	"github.com/stackrox/stackrox/pkg/httputil/proxy"
	"github.com/stackrox/stackrox/pkg/logging"
	"google.golang.org/grpc"
)

var (
	log = logging.LoggerForModule()

	x509Err = x509.UnknownAuthorityError{}.Error()
)

type serviceImpl struct {
	sensorConnectionManager connection.Manager
	client                  http.Client
}

func (s *serviceImpl) ReconciliationStatsByCluster(context.Context, *central.Empty) (*central.ReconciliationStatsByClusterResponse, error) {
	var resp central.ReconciliationStatsByClusterResponse
	connections := s.sensorConnectionManager.GetActiveConnections()
	for _, conn := range connections {
		deletionsByTyp, reconciliationDone := conn.ObjectsDeletedByReconciliation()
		var convertedDeletions map[string]int32
		if reconciliationDone {
			convertedDeletions = make(map[string]int32, len(deletionsByTyp))
			for k, v := range deletionsByTyp {
				convertedDeletions[k] = int32(v)
			}
		}
		resp.Stats = append(resp.Stats, &central.ReconciliationStatsByClusterResponse_ReconciliationStatsForCluster{
			ClusterId:            conn.ClusterID(),
			ReconciliationDone:   reconciliationDone,
			DeletedObjectsByType: convertedDeletions,
		})
	}
	return &resp, nil
}

func (s *serviceImpl) URLHasValidCert(ctx context.Context, req *central.URLHasValidCertRequest) (*central.URLHasValidCertResponse, error) {
	if !strings.HasPrefix(req.GetUrl(), "https://") {
		return nil, errors.Wrapf(errox.InvalidArgs, "url %q must start with https", req.GetUrl())
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

func (s *serviceImpl) EnvVars(ctx context.Context, _ *central.Empty) (*central.EnvVarsResponse, error) {
	envVars := os.Environ()
	return &central.EnvVarsResponse{
		EnvVars: envVars,
	}, nil
}

func (s *serviceImpl) RandomData(ctx context.Context, req *central.RandomDataRequest) (*central.RandomDataResponse, error) {
	resp := &central.RandomDataResponse{
		Data: make([]byte, req.GetSize_()),
	}

	_, _ = rand.Read(resp.Data)
	return resp, nil
}

// New creates a new Service.
func New(sensorConnectionManager connection.Manager) Service {
	return &serviceImpl{
		sensorConnectionManager: sensorConnectionManager,
		client: http.Client{
			Timeout:   20 * time.Second,
			Transport: proxy.RoundTripper(),
		},
	}
}

func (s *serviceImpl) RegisterServiceServer(server *grpc.Server) {
	central.RegisterDevelopmentServiceServer(server, s)
}

func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return central.RegisterDevelopmentServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, allow.Anonymous().Authorized(ctx, fullMethodName)
}
