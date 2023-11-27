package service

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"strings"

	"github.com/gogo/protobuf/types"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	imageIntegrationStore "github.com/stackrox/rox/central/imageintegration/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/tlsutils"
	"github.com/stackrox/rox/pkg/urlfmt"
	"github.com/stackrox/rox/pkg/utils"
	"google.golang.org/grpc"
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.Authenticated(): {
			"/v1.CredentialExpiryService/GetCertExpiry",
		},
	})
)

// ClusterService is the struct that manages the cluster API
type serviceImpl struct {
	v1.UnimplementedCredentialExpiryServiceServer

	imageIntegrations      imageIntegrationStore.DataStore
	scannerConfig          *tls.Config
	scannerV4IndexerConfig *tls.Config
}

type endpointWithConfig struct {
	endpoint  string
	tlsConfig *tls.Config
	subject   mtls.Subject
}

func (s *serviceImpl) GetCertExpiry(ctx context.Context, request *v1.GetCertExpiry_Request) (*v1.GetCertExpiry_Response, error) {
	switch request.GetComponent() {
	case v1.GetCertExpiry_CENTRAL:
		return s.getCentralCertExpiry()
	case v1.GetCertExpiry_SCANNER:
		return s.getScannerCertExpiry(ctx)
	}
	return nil, errors.Wrapf(errox.InvalidArgs, "invalid component: %v", request.GetComponent())
}

func (s *serviceImpl) getCentralCertExpiry() (*v1.GetCertExpiry_Response, error) {
	cert, err := mtls.LeafCertificateFromFile()
	if err != nil {
		return nil, errors.Errorf("failed to retrieve leaf certificate: %v", err)
	}
	if len(cert.Certificate) == 0 {
		return nil, errors.New("no central cert found")
	}
	parsedCert, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return nil, errors.New("failed to parse central cert")
	}
	expiry, err := types.TimestampProto(parsedCert.NotAfter)
	if err != nil {
		return nil, errors.Errorf("failed to convert timestamp: %v", err)
	}
	return &v1.GetCertExpiry_Response{Expiry: expiry}, nil
}

// ensureTLSAndReturnAddr returns an address from endpoint that can be passed to tls.Dial,
// or an error if the endpoint does not represent a valid TLS server.
func ensureTLSAndReturnAddr(endpoint string) (string, error) {
	if !strings.HasPrefix(endpoint, "https://") {
		return "", errors.Errorf("endpoint %s is not an HTTPS endpoint", endpoint)
	}
	server := urlfmt.GetServerFromURL(endpoint)
	if server == "" {
		return "", errors.Errorf("failed to retrieve server from endpoint %s", endpoint)
	}
	if strings.Contains(server, ":") {
		return server, nil
	}
	return fmt.Sprintf("%s:443", server), nil
}

func (s *serviceImpl) maybeGetExpiryFomScannerAt(ctx context.Context, epWithCfg endpointWithConfig) (*types.Timestamp, error) {
	addr, err := ensureTLSAndReturnAddr(epWithCfg.endpoint)
	if err != nil {
		return nil, err
	}
	conn, err := tlsutils.DialContext(ctx, "tcp", addr, epWithCfg.tlsConfig)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to contact scanner at %s", epWithCfg.endpoint)
	}
	defer utils.IgnoreError(conn.Close)
	certs := conn.ConnectionState().PeerCertificates
	if len(certs) == 0 {
		return nil, errors.Errorf("scanner at %s returned no peer certs", epWithCfg.endpoint)
	}
	leafCert := certs[0]
	if cn := leafCert.Subject.CommonName; cn != epWithCfg.subject.CN() {
		return nil, errors.Errorf("common name of scanner at %s (%s) is not as expected", epWithCfg.endpoint, cn)
	}
	expiry, err := types.TimestampProto(leafCert.NotAfter)
	if err != nil {
		return nil, errors.Wrap(err, "converting timestamp")
	}
	return expiry, nil
}

func (s *serviceImpl) getScannerCertExpiry(ctx context.Context) (*v1.GetCertExpiry_Response, error) {
	if s.scannerConfig == nil {
		return nil, errors.New("could not load TLS config to talk to scanner")
	}
	integrations, err := s.imageIntegrations.GetImageIntegrations(ctx, &v1.GetImageIntegrationsRequest{})
	if err != nil {
		return nil, errors.Errorf("failed to retrieve image integrations: %v", err)
	}

	var endpoints []endpointWithConfig

	// Here we collect the endpoints for scanner V2 and for scanner V4 integrations.
	for _, integration := range integrations {
		if clairify := integration.GetClairify(); clairify != nil {
			endpoints = append(endpoints, endpointWithConfig{
				endpoint:  clairify.GetEndpoint(),
				tlsConfig: s.scannerConfig,
				subject:   mtls.ScannerSubject,
			})
		} else if clairV4 := integration.GetClairV4(); clairV4 != nil {
			endpoints = append(endpoints, endpointWithConfig{
				endpoint:  clairV4.GetEndpoint(),
				tlsConfig: s.scannerV4IndexerConfig,
				subject:   mtls.ScannerV4IndexerSubject,
			})
		}
	}
	if len(endpoints) == 0 {
		return nil, errors.Wrap(errox.InvalidArgs, "StackRox Scanner is not integrated")
	}
	errC := make(chan error, len(endpoints))
	expiryC := make(chan *types.Timestamp, len(endpoints))
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	for _, endpoint := range endpoints {
		go func(endpoint endpointWithConfig) {
			expiry, err := s.maybeGetExpiryFomScannerAt(ctx, endpoint)
			if err != nil {
				errC <- err
				return
			}
			expiryC <- expiry
		}(endpoint)
	}

	errorList := errorhelpers.NewErrorList("failed to determine scanner cert expiry")
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case err := <-errC:
			errorList.AddError(err)
			// All the endpoints have failed.
			if len(errorList.ErrorStrings()) == len(endpoints) {
				return nil, errors.New(errorList.String())
			}
		case expiry := <-expiryC:
			return &v1.GetCertExpiry_Response{Expiry: expiry}, nil
		}
	}
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterCredentialExpiryServiceServer(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterCredentialExpiryServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}
