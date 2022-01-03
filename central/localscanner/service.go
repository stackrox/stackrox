package localscanner

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
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

func localCertificatesForCertMap(serviceType storage.ServiceType, certificates secretDataMap) *central.LocalScannerCertificates {
	// FIXME replace secretDataMap in central/localscanner/certificates.go by typed struct
	return &central.LocalScannerCertificates{
		ServiceType: serviceType,
		Ca:          certificates[mtls.CACertFileName],
		Cert:        certificates[mtls.ServiceCertFileName],
		Key:         certificates[mtls.ServiceKeyFileName],
	}
}

func (s *serviceImpl) IssueLocalScannerCerts(_ context.Context, request *central.IssueLocalScannerCertsRequest) (*central.IssueLocalScannerCertsResponse, error) {
	if request.GetNamespace() == "" {
		return nil, errors.New("namespace is required to issue the certificates for the local scanner")
	}
	if request.GetClusterId() == "" {
		return nil, errors.New("cluster id is required to issue the certificates for the local scanner")
	}

	scannerCertificates, err := generateServiceCertMap(storage.ServiceType_SCANNER_SERVICE, request.GetNamespace(), request.GetClusterId())
	errorFormat := "error generating certificate for service %s"
	if err != nil {
		return nil, errors.Wrapf(err, errorFormat, storage.ServiceType_SCANNER_SERVICE)
	}
	scannerDBCertificates, err := generateServiceCertMap(storage.ServiceType_SCANNER_DB_SERVICE, request.GetNamespace(), request.GetClusterId())
	if err != nil {
		return nil, errors.Wrapf(err, errorFormat, storage.ServiceType_SCANNER_DB_SERVICE)
	}

	return &central.IssueLocalScannerCertsResponse{
		ScannerCerts:   localCertificatesForCertMap(storage.ServiceType_SCANNER_SERVICE, scannerCertificates),
		ScannerDbCerts: localCertificatesForCertMap(storage.ServiceType_SCANNER_DB_SERVICE, scannerDBCertificates),
	}, nil
}
