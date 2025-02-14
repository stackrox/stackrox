package service

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sort"
	"strings"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/pkg/errors"
	iiDStore "github.com/stackrox/rox/central/imageintegration/datastore"
	iiStore "github.com/stackrox/rox/central/imageintegration/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/postgres/pgconfig"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/scanners/scannerv4"
	"github.com/stackrox/rox/pkg/tlsutils"
	"github.com/stackrox/rox/pkg/urlfmt"
	"github.com/stackrox/rox/pkg/utils"
	"google.golang.org/grpc"
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.Authenticated(): {
			v1.CredentialExpiryService_GetCertExpiry_FullMethodName,
		},
	})
)

// ClusterService is the struct that manages the cluster API
type serviceImpl struct {
	v1.UnimplementedCredentialExpiryServiceServer

	imageIntegrations iiDStore.DataStore
	scannerConfigs    map[mtls.Subject]*tls.Config
	expiryFunc        func(ctx context.Context, subject mtls.Subject, tlsConfig *tls.Config, endpoint string) (*time.Time, error)
}

func (s *serviceImpl) GetCertExpiry(ctx context.Context, request *v1.GetCertExpiry_Request) (*v1.GetCertExpiry_Response, error) {
	switch request.GetComponent() {
	case v1.GetCertExpiry_CENTRAL:
		return s.getCentralCertExpiry()
	case v1.GetCertExpiry_SCANNER:
		return s.getScannerCertExpiry(ctx)
	case v1.GetCertExpiry_SCANNER_V4:
		return s.getScannerV4CertExpiry(ctx)
	case v1.GetCertExpiry_CENTRAL_DB:
		return s.getCentralDBCertExpiry()
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
	expiry, err := protocompat.ConvertTimeToTimestampOrError(parsedCert.NotAfter)
	if err != nil {
		return nil, errors.Errorf("failed to convert timestamp: %v", err)
	}
	return &v1.GetCertExpiry_Response{Expiry: expiry}, nil
}

func (s *serviceImpl) getCentralDBCertExpiry() (*v1.GetCertExpiry_Response, error) {
	if pgconfig.IsExternalDatabase() {
		return nil, nil
	}
	pgConfigMap, _, err := pgconfig.GetPostgresConfig()
	if err != nil {
		return nil, errors.Wrap(err, "Error reading central db config")
	}
	if pgConfigMap == nil {
		return nil, errors.Wrap(errox.NotFound, "Central db config not found")
	}

	host, ok := pgConfigMap["host"]
	if !ok {
		return nil, errors.Wrap(errox.InvalidArgs, "'host' parameter not defined in central db config")
	}
	port, ok := pgConfigMap["port"]
	if !ok {
		return nil, errors.Wrap(errox.InvalidArgs, "'port' parameter not defined in central db config")
	}
	endpoint := fmt.Sprintf("%s:%s", host, port)

	conn, err := net.Dial("tcp", endpoint)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to dial central db on endpoint '%s'", endpoint)
	}
	defer utils.IgnoreError(conn.Close)

	tlsConfig, err := clientconn.TLSConfig(mtls.CentralDBSubject, clientconn.TLSConfigOptions{
		InsecureSkipVerify: true,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to initialize TLS config for %q", mtls.CentralDBSubject.Identifier)
	}
	tlsConn, err := tlsConnectToCentralDB(conn, tlsConfig)
	if err != nil {
		return nil, err
	}

	certs := tlsConn.ConnectionState().PeerCertificates
	if len(certs) == 0 {
		return nil, errors.Errorf("%q at %s returned no peer certs", mtls.CentralDBSubject.Identifier, endpoint)
	}
	leafCert := certs[0]
	if leafCert == nil {
		return nil, nil
	}
	if cn := leafCert.Subject.CommonName; cn != mtls.CentralDBSubject.CN() {
		return nil, errors.Errorf("common name of %q at %s (%s) is not as expected", mtls.CentralDBSubject.Identifier, endpoint, cn)
	}
	if leafCert.NotAfter.IsZero() {
		return nil, nil
	}
	certExpiry, err := protocompat.ConvertTimeToTimestampOrError(leafCert.NotAfter)
	if err != nil {
		return nil, err
	}
	return &v1.GetCertExpiry_Response{Expiry: certExpiry}, nil
}

// tlsConnectToCentralDB implements the protocol to establish a TLS connection to a postgres database server
func tlsConnectToCentralDB(conn net.Conn, tlsConfig *tls.Config) (*tls.Conn, error) {
	err := binary.Write(conn, binary.BigEndian, []int32{8, 80877103})
	if err != nil {
		return nil, errors.Wrap(err, "Failed to send initiation message to central db")
	}
	response := make([]byte, 1)
	if _, err = io.ReadFull(conn, response); err != nil {
		return nil, errors.Wrap(err, "Failed to receive a reply from central db")
	}
	if response[0] != 'S' {
		return nil, errors.New("Central db refused TLS connection")
	}
	client := tls.Client(conn, tlsConfig)
	err = client.Handshake()
	if err != nil {
		return nil, errors.Wrap(err, "Failed TLS handshake with central db")
	}
	return client, nil
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

func maybeGetExpiryFromScannerAt(ctx context.Context, subject mtls.Subject, tlsConfig *tls.Config, endpoint string) (*time.Time, error) {
	conn, err := tlsutils.DialContext(ctx, "tcp", endpoint, tlsConfig)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to contact scanner at %s", endpoint)
	}
	defer utils.IgnoreError(conn.Close)
	certs := conn.ConnectionState().PeerCertificates
	if len(certs) == 0 {
		return nil, errors.Errorf("%q at %s returned no peer certs", subject.Identifier, endpoint)
	}
	leafCert := certs[0]
	if cn := leafCert.Subject.CommonName; cn != subject.CN() {
		return nil, errors.Errorf("common name of %q at %s (%s) is not as expected", subject.Identifier, endpoint, cn)
	}
	return &leafCert.NotAfter, nil
}

func (s *serviceImpl) getScannerCertExpiry(ctx context.Context) (*v1.GetCertExpiry_Response, error) {
	scannerConfig := s.scannerConfigs[mtls.ScannerSubject]
	if scannerConfig == nil {
		return nil, errors.New("could not load TLS config to talk to scanner")
	}
	integrations, err := s.imageIntegrations.GetImageIntegrations(ctx, &v1.GetImageIntegrationsRequest{})
	if err != nil {
		return nil, errors.Errorf("failed to retrieve image integrations: %v", err)
	}

	var clairifyEndpoints []string
	for _, integration := range integrations {
		if clairify := integration.GetClairify(); clairify != nil {
			clairifyEndpoints = append(clairifyEndpoints, clairify.GetEndpoint())
		}
	}
	if len(clairifyEndpoints) == 0 {
		return nil, errors.Wrap(errox.InvalidArgs, "StackRox Scanner is not integrated")
	}
	errC := make(chan error, len(clairifyEndpoints))
	expiryC := make(chan *time.Time, len(clairifyEndpoints))
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	for _, endpoint := range clairifyEndpoints {
		go func(endpoint string) {
			addr, err := ensureTLSAndReturnAddr(endpoint)
			if err != nil {
				errC <- err
				return
			}
			expiry, err := s.expiryFunc(ctx, mtls.ScannerSubject, scannerConfig, addr)
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
			if len(errorList.ErrorStrings()) == len(clairifyEndpoints) {
				return nil, errorList.ToError()
			}
		case expiry := <-expiryC:
			if expiry == nil {
				return &v1.GetCertExpiry_Response{Expiry: nil}, nil
			}
			certExpiry, err := protocompat.ConvertTimeToTimestampOrError(*expiry)
			if err != nil {
				return nil, err
			}
			return &v1.GetCertExpiry_Response{Expiry: certExpiry}, nil
		}
	}
}

func (s *serviceImpl) getScannerV4CertExpiry(ctx context.Context) (*v1.GetCertExpiry_Response, error) {
	if !features.ScannerV4.Enabled() {
		return nil, errors.Wrap(errox.InvalidArgs, "Scanner V4 is not enabled/integrated")
	}
	indexerConfig := s.scannerConfigs[mtls.ScannerV4IndexerSubject]
	matcherConfig := s.scannerConfigs[mtls.ScannerV4MatcherSubject]
	if indexerConfig == nil && matcherConfig == nil {
		return nil, errors.New("could not load TLS configs to talk to Scanner V4 indexer and matcher")
	}

	integration, exists, err := s.imageIntegrations.GetImageIntegration(ctx, iiStore.DefaultScannerV4Integration.GetId())
	if err != nil {
		return nil, errors.Wrap(err, "failed to retrieve Scanner V4 image integration")
	}
	if !exists {
		return nil, errors.New("Scanner V4 image integration missing")
	}

	s4Config := integration.GetScannerV4()
	indexerEndpoint := scannerv4.DefaultIndexerEndpoint
	if endpoint := s4Config.GetIndexerEndpoint(); endpoint != "" {
		indexerEndpoint = endpoint
	}
	matcherEndpoint := scannerv4.DefaultMatcherEndpoint
	if endpoint := s4Config.GetMatcherEndpoint(); endpoint != "" {
		matcherEndpoint = endpoint
	}

	numEndpoints := 2
	errC := make(chan error, numEndpoints)
	expiryC := make(chan *time.Time, numEndpoints)
	getExpiry := func(subject mtls.Subject, endpoint string) {
		expiry, err := s.expiryFunc(ctx, subject, s.scannerConfigs[subject], endpoint)
		if err != nil {
			errC <- err
			return
		}

		log.Debugf("Obtained expiry for %q: %s", subject.Identifier, expiry)
		expiryC <- expiry
	}

	go getExpiry(mtls.ScannerV4IndexerSubject, indexerEndpoint)
	go getExpiry(mtls.ScannerV4MatcherSubject, matcherEndpoint)

	errorList := errorhelpers.NewErrorList("failed to determine Scanner V4 cert expiry")
	expiries := make([]*time.Time, 0, numEndpoints)
	for i := 0; i < numEndpoints; i++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case err := <-errC:
			errorList.AddError(err)
		case expiry := <-expiryC:
			expiries = append(expiries, expiry)
		}
	}
	if len(expiries) == 0 {
		return nil, errorList.ToError()
	}

	sort.Slice(expiries, func(i, j int) bool {
		if expiries[i] == nil {
			return true
		}
		if expiries[j] == nil {
			return false
		}
		return expiries[i].Compare(*expiries[j]) < 0
	})

	if expiries[0] == nil {
		return &v1.GetCertExpiry_Response{Expiry: nil}, nil
	}
	certExpiry, err := protocompat.ConvertTimeToTimestampOrError(*expiries[0])
	if err != nil {
		return nil, err
	}

	return &v1.GetCertExpiry_Response{Expiry: certExpiry}, nil
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
