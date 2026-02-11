package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	clusterCVEDatastore "github.com/stackrox/rox/central/cve/cluster/datastore"
	imageCVEDatastore "github.com/stackrox/rox/central/cve/image/v2/datastore"
	nodeCVEDatastore "github.com/stackrox/rox/central/cve/node/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/set"
	"google.golang.org/grpc"
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.Image), permissions.View(resources.Node), permissions.View(resources.Cluster)): {
			v1.CVEService_GetCVEMetadata_FullMethodName,
		},
	})
)

// serviceImpl provides APIs for CVE metadata.
type serviceImpl struct {
	v1.UnimplementedCVEServiceServer

	imageCVEs   imageCVEDatastore.DataStore
	nodeCVEs    nodeCVEDatastore.DataStore
	clusterCVEs clusterCVEDatastore.DataStore
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterCVEServiceServer(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterCVEServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the auth criteria for this service.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

// GetCVEMetadata returns CVE metadata for the specified CVE IDs.
func (s *serviceImpl) GetCVEMetadata(ctx context.Context, req *v1.GetCVEMetadataRequest) (*v1.GetCVEMetadataResponse, error) {
	cves := make(map[string]*v1.GetCVEMetadataResponse_CVEMetadata)

	for _, cveID := range req.GetCveIds() {
		metadata := &v1.GetCVEMetadataResponse_CVEMetadata{}
		cvssScores := set.NewSet[*storage.CVSSScore]()
		types := set.NewSet[storage.CVE_CVEType]()

		// Try to get from image CVE datastore.
		if imageCVE, found, err := s.imageCVEs.Get(ctx, cveID); err == nil && found {
			metadata.Severity = imageCVE.GetSeverity()
			if imageCVE.GetCveBaseInfo() != nil {
				metadata.Summary = imageCVE.GetCveBaseInfo().GetSummary()
				metadata.Link = imageCVE.GetCveBaseInfo().GetLink()

				for _, score := range imageCVE.GetCveBaseInfo().GetCvssMetrics() {
					cvssScores.Add(score)
				}
			}
			types.Add(storage.CVE_IMAGE_CVE)
		}

		// Try to get from node CVE datastore.
		if nodeCVE, found, err := s.nodeCVEs.Get(ctx, cveID); err == nil && found {
			if metadata.GetSeverity() == storage.VulnerabilitySeverity_UNKNOWN_VULNERABILITY_SEVERITY {
				metadata.Severity = nodeCVE.GetSeverity()
			}
			if nodeCVE.GetCveBaseInfo() != nil {
				if metadata.GetSummary() == "" {
					metadata.Summary = nodeCVE.GetCveBaseInfo().GetSummary()
				}
				if metadata.GetLink() == "" {
					metadata.Link = nodeCVE.GetCveBaseInfo().GetLink()
				}

				for _, score := range nodeCVE.GetCveBaseInfo().GetCvssMetrics() {
					cvssScores.Add(score)
				}
			}
			types.Add(storage.CVE_NODE_CVE)
		}

		// Try to get from cluster CVE datastore (covers K8S and Istio).
		if clusterCVE, found, err := s.clusterCVEs.Get(ctx, cveID); err == nil && found {
			if metadata.GetSeverity() == storage.VulnerabilitySeverity_UNKNOWN_VULNERABILITY_SEVERITY {
				metadata.Severity = clusterCVE.GetSeverity()
			}
			if clusterCVE.GetCveBaseInfo() != nil {
				if metadata.GetSummary() == "" {
					metadata.Summary = clusterCVE.GetCveBaseInfo().GetSummary()
				}
				if metadata.GetLink() == "" {
					metadata.Link = clusterCVE.GetCveBaseInfo().GetLink()
				}

				for _, score := range clusterCVE.GetCveBaseInfo().GetCvssMetrics() {
					cvssScores.Add(score)
				}
			}

			if clusterCVE.GetType() == storage.CVE_K8S_CVE {
				types.Add(storage.CVE_K8S_CVE)
			} else if clusterCVE.GetType() == storage.CVE_ISTIO_CVE {
				types.Add(storage.CVE_ISTIO_CVE)
			}
		}

		// Only add to response if we found the CVE in at least one datastore.
		if cvssScores.Cardinality() > 0 || types.Cardinality() > 0 {
			metadata.CvssScores = cvssScores.AsSlice()
			metadata.Types = types.AsSlice()
			cves[cveID] = metadata
		}
	}

	return &v1.GetCVEMetadataResponse{Cves: cves}, nil
}
