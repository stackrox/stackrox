package localscanner

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/errorhelpers"
	pkgGRPC "github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/mtls"
	"google.golang.org/grpc"
)

// Service is the interface for the local scanner service.
type Service interface {
	pkgGRPC.APIService
	central.LocalScannerServiceServer
}

// New creates a new local scanner service.
func New(clusters clusterDataStore.DataStore) Service {
	return &serviceImpl{
		clusters: clusters,
	}
}

type serviceImpl struct {
	clusters clusterDataStore.DataStore
}

func (s *serviceImpl) RegisterServiceServer(server *grpc.Server) {
	central.RegisterLocalScannerServiceServer(server, s)
}

func (s *serviceImpl) RegisterServiceHandler(context.Context, *runtime.ServeMux, *grpc.ClientConn) error {
	return nil
}

func (s *serviceImpl) IssueLocalScannerCerts(ctx context.Context, request *central.IssueLocalScannerCertsRequest) (*central.IssueLocalScannerCertsResponse, error) {
	clusterID, err := s.authorizeAndGetClusterID(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failure fetching cluster ID")
	}

	return issueLocalScannerCerts(request.GetNamespace(), clusterID)
}

func (s *serviceImpl) authorizeAndGetClusterID(ctx context.Context) (string, error) {
	identity, err := authn.IdentityFromContext(ctx)
	if err != nil {
		return "", errors.Wrap(err, "could not determine identity from request context")
	}
	if identity == nil {
		return "", errors.New("could not determine identity from request context")
	}

	svc := identity.Service()
	if svc == nil || svc.GetType() != storage.ServiceType_SENSOR_SERVICE {
		return "", errorhelpers.NewErrNotAuthorized("only sensor may access this API")
	}

	clusterID := svc.GetId()
	if centralsensor.IsInitCertClusterID(clusterID) {
		return "", errors.Errorf("cannot issue local Scanner credentials for a cluster that has not yet been assigned an ID: found id %q", clusterID)
	}

	_, clusterExists, err := s.clusters.GetCluster(ctx, clusterID)
	if err != nil {
		return "", errors.Wrapf(err, "error fetching cluster with ID %q", clusterID)
	}
	if !clusterExists {
		return "", errors.Errorf("cluster with ID %q does not exist", clusterID)
	}

	return clusterID, nil
}

func issueLocalScannerCerts(namespace string, clusterID string) (*central.IssueLocalScannerCertsResponse, error) {
	if namespace == "" {
		return nil, errors.New("namespace is required to issue the certificates for the local scanner")
	}

	var certIssueError error
	scannerCertificates, err := localScannerCertificatesFor(storage.ServiceType_SCANNER_SERVICE, namespace, clusterID)
	if err != nil {
		certIssueError = multierror.Append(certIssueError, err)
	}
	scannerDBCertificates, err := localScannerCertificatesFor(storage.ServiceType_SCANNER_DB_SERVICE, namespace, clusterID)
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

func localScannerCertificatesFor(serviceType storage.ServiceType, namespace string, clusterID string) (*central.LocalScannerCertificates, error) {
	certificates, err := generateServiceCertMap(serviceType, namespace, clusterID)
	if err != nil {
		return nil, errors.Wrapf(err, "error generating certificate for service %s", serviceType)
	}

	return &central.LocalScannerCertificates{
		Ca:   certificates[mtls.CACertFileName],
		Cert: certificates[mtls.ServiceCertFileName],
		Key:  certificates[mtls.ServiceKeyFileName],
	}, nil
}
