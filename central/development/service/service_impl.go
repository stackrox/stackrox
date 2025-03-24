package service

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/pkg/errors"
	imageDatastore "github.com/stackrox/rox/central/image/datastore"
	riskManager "github.com/stackrox/rox/central/risk/manager"
	"github.com/stackrox/rox/central/sensor/service/connection"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/defaults/accesscontrol"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/httputil/proxy"
	"github.com/stackrox/rox/pkg/random"
	"google.golang.org/grpc"
)

var (
	x509Err = x509.UnknownAuthorityError{}.Error()
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.WithRole(accesscontrol.Admin): {
			"/central.DevelopmentService/ReplicateImage",
			"/central.DevelopmentService/URLHasValidCert",
			"/central.DevelopmentService/RandomData",
			"/central.DevelopmentService/EnvVars",
			"/central.DevelopmentService/ReconciliationStatsByCluster",
		},
	})
)

// New creates a new Service.
func New(sensorConnectionManager connection.Manager, imageDatastore imageDatastore.DataStore, riskManager riskManager.Manager) Service {
	client := retryablehttp.NewClient()
	client.HTTPClient = &http.Client{
		Timeout:   20 * time.Second,
		Transport: proxy.RoundTripper(),
	}
	return &serviceImpl{
		imageDatastore:          imageDatastore,
		riskManager:             riskManager,
		sensorConnectionManager: sensorConnectionManager,
		client:                  client.StandardClient(),
	}
}

type serviceImpl struct {
	central.UnimplementedDevelopmentServiceServer

	sensorConnectionManager connection.Manager
	imageDatastore          imageDatastore.DataStore
	riskManager             riskManager.Manager
	client                  *http.Client
}

func (s *serviceImpl) ReplicateImage(ctx context.Context, req *central.ReplicateImageRequest) (*central.Empty, error) {
	image, exists, err := s.imageDatastore.GetImage(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.Errorf("image %q does not exist", req.GetId())
	}
	for i := 0; i < int(req.GetTimes()); i++ {
		image.Id = random.GenerateString(65, random.HexValues)
		if err := s.riskManager.CalculateRiskAndUpsertImage(image); err != nil {
			return nil, err
		}
	}
	return &central.Empty{}, nil
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

func (s *serviceImpl) URLHasValidCert(_ context.Context, req *central.URLHasValidCertRequest) (*central.URLHasValidCertResponse, error) {
	u, err := url.Parse(req.GetUrl())
	if err != nil {
		return nil, errors.Wrapf(errox.InvalidArgs, "invalid url %s", err.Error())
	}

	// Since we are using http.DefaultHttpClient, it relies on the same CA certificates as x509.SystemCertPool.
	// This means that verifying the provided certificate is equivalent to making a call to a service;
	// we are primarily interested in the certificate validation process.
	// Certificates are installed by placing them in TRUSTED_CA_FILE as detailed in:
	// https://github.com/stackrox/stackrox/blob/4.4.0/tests/e2e/run.sh#L97
	// The certificates are then copied to a secret, mounted at `/usr/local/share/ca-certificates/`,
	// and installed using `update-ca-certificates`, as described in:
	// https://github.com/stackrox/stackrox/blob/4.4.0/image/templates/helm/stackrox-central/templates/_init.tpl.htpl#L208
	// Consequently, they will be located in `/etc/ssl/certs/ca-certificates.crt`,
	// which is the default CA path for Go, as specified in:
	// https://github.com/golang/go/blob/ad77cefeb2f5b3f1cef4383e974195ffc8610236/src/crypto/x509/root_linux.go#L11
	if req.CertPEM == "" {
		_, err = s.client.Head(req.GetUrl())
	} else {
		err = verifyProvidedCert(req, u)
	}
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

func verifyProvidedCert(req *central.URLHasValidCertRequest, u *url.URL) error {
	block, _ := pem.Decode([]byte(req.GetCertPEM()))
	if block == nil {
		return errors.Wrap(errox.InvalidArgs, "failed to decode certificate")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return errors.Wrapf(errox.InvalidArgs, "failed to parse certificate %s", err.Error())
	}
	pool, err := x509.SystemCertPool()
	if err != nil {
		return errors.Wrapf(errox.ServerError, "failed to get system cert pool %s", err.Error())
	}

	opts := x509.VerifyOptions{
		DNSName: u.Host,
		Roots:   pool,
	}

	_, err = cert.Verify(opts)
	return err
}

func (s *serviceImpl) EnvVars(_ context.Context, _ *central.Empty) (*central.EnvVarsResponse, error) {
	envVars := os.Environ()
	return &central.EnvVarsResponse{
		EnvVars: envVars,
	}, nil
}

func (s *serviceImpl) RandomData(_ context.Context, req *central.RandomDataRequest) (*central.RandomDataResponse, error) {
	resp := &central.RandomDataResponse{
		Data: make([]byte, req.GetSize()),
	}

	_, _ = rand.Read(resp.Data)
	return resp, nil
}

func (s *serviceImpl) RegisterServiceServer(server *grpc.Server) {
	central.RegisterDevelopmentServiceServer(server, s)
}

func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return central.RegisterDevelopmentServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}
