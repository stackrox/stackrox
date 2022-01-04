package localscanner

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	pkgGRPC "github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/mtls"
	"google.golang.org/grpc"
)

// Service is the interface for the local scanner service.
type Service interface {
	pkgGRPC.APIService
	central.LocalScannerServiceServer
}

// New creates a new local scanner service.
func New() Service {
	return &serviceImpl{}
}

type serviceImpl struct{}

func (s *serviceImpl) RegisterServiceServer(server *grpc.Server) {
	central.RegisterLocalScannerServiceServer(server, s)
}

func (s *serviceImpl) RegisterServiceHandler(context.Context, *runtime.ServeMux, *grpc.ClientConn) error {
	return nil
}

func localScannerCertificatesFor(serviceType storage.ServiceType, namespace string, clusterID string) (*central.LocalScannerCertificates, error) {
	certificates, err := generateServiceCertMap(serviceType, namespace, clusterID)
	if err != nil {
		return nil, errors.Wrapf(err, "error generating certificate for service %s", serviceType)
	}

	return &central.LocalScannerCertificates{
		ServiceType: serviceType,
		Ca:          certificates[mtls.CACertFileName],
		Cert:        certificates[mtls.ServiceCertFileName],
		Key:         certificates[mtls.ServiceKeyFileName],
	}, nil
}

func (s *serviceImpl) IssueLocalScannerCerts(_ context.Context, request *central.IssueLocalScannerCertsRequest) (*central.IssueLocalScannerCertsResponse, error) {
	if request.GetNamespace() == "" {
		return nil, errors.New("namespace is required to issue the certificates for the local scanner")
	}
	if request.GetClusterId() == "" {
		return nil, errors.New("cluster id is required to issue the certificates for the local scanner")
	}

	var certIssueError error
	scannerCertificates, err := localScannerCertificatesFor(storage.ServiceType_SCANNER_SERVICE, request.GetNamespace(), request.GetClusterId())
	if err != nil {
		certIssueError = multierror.Append(certIssueError, err)
	}
	scannerDBCertificates, err := localScannerCertificatesFor(storage.ServiceType_SCANNER_DB_SERVICE, request.GetNamespace(), request.GetClusterId())
	if err != nil {
		certIssueError = multierror.Append(certIssueError, err)
	}
	if certIssueError != nil {
		return nil, certIssueError
	}

	return &central.IssueLocalScannerCertsResponse{
		ScannerCerts:   scannerCertificates,
		ScannerDbCerts: scannerDBCertificates,
	}, nil
}
