package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/stackrox/rox/central/convert/storagetov2"
	clusterCVEDatastore "github.com/stackrox/rox/central/cve/cluster/datastore"
	imageCVEDatastore "github.com/stackrox/rox/central/cve/image/v2/datastore"
	nodeCVEDatastore "github.com/stackrox/rox/central/cve/node/datastore"
	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/set"
	"google.golang.org/grpc"
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.Image), permissions.View(resources.Node), permissions.View(resources.Cluster)): {
			v2.CVEMetadataService_ListCVEMetadata_FullMethodName,
		},
	})
	log = logging.LoggerForModule()
)

// serviceImpl provides APIs for CVE metadata.
type serviceImpl struct {
	v2.UnimplementedCVEMetadataServiceServer

	imageCVEs   imageCVEDatastore.DataStore
	nodeCVEs    nodeCVEDatastore.DataStore
	clusterCVEs clusterCVEDatastore.DataStore
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v2.RegisterCVEMetadataServiceServer(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v2.RegisterCVEMetadataServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the auth criteria for this service.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

// ListCVEMetadata returns CVE metadata for the specified CVE IDs.
func (s *serviceImpl) ListCVEMetadata(ctx context.Context, req *v2.ListCVEMetadataRequest) (*v2.ListCVEMetadataResponse, error) {
	cves := make(map[string]*v2.EmbeddedVulnerability)

	for _, cveID := range req.GetCveIds() {
		vuln := &v2.EmbeddedVulnerability{
			Cve: cveID,
		}
		cvssScores := set.NewSet[*storage.CVSSScore]()
		foundCVE := false

		// Try to get from image CVE datastore.
		if imageCVE, found, err := s.imageCVEs.Get(ctx, cveID); err == nil && found {
			foundCVE = true
			vuln.Severity = storagetov2.ConvertVulnerabilitySeverity(imageCVE.GetSeverity())
			vuln.Cvss = imageCVE.GetCvss()
			if imageCVE.GetCveBaseInfo() != nil {
				vuln.Summary = imageCVE.GetCveBaseInfo().GetSummary()
				vuln.Link = imageCVE.GetCveBaseInfo().GetLink()
				vuln.PublishedOn = imageCVE.GetCveBaseInfo().GetPublishedOn()
				vuln.LastModified = imageCVE.GetCveBaseInfo().GetLastModified()

				for _, score := range imageCVE.GetCveBaseInfo().GetCvssMetrics() {
					cvssScores.Add(score)
				}
			}
		}

		// Try to get from node CVE datastore.
		if nodeCVE, found, err := s.nodeCVEs.Get(ctx, cveID); err == nil && found {
			foundCVE = true
			if vuln.GetSeverity() == v2.VulnerabilitySeverity_UNKNOWN_VULNERABILITY_SEVERITY {
				vuln.Severity = storagetov2.ConvertVulnerabilitySeverity(nodeCVE.GetSeverity())
			}
			if vuln.GetCvss() == 0 {
				vuln.Cvss = nodeCVE.GetCvss()
			}
			if nodeCVE.GetCveBaseInfo() != nil {
				if vuln.GetSummary() == "" {
					vuln.Summary = nodeCVE.GetCveBaseInfo().GetSummary()
				}
				if vuln.GetLink() == "" {
					vuln.Link = nodeCVE.GetCveBaseInfo().GetLink()
				}
				if vuln.GetPublishedOn() == nil {
					vuln.PublishedOn = nodeCVE.GetCveBaseInfo().GetPublishedOn()
				}
				if vuln.GetLastModified() == nil {
					vuln.LastModified = nodeCVE.GetCveBaseInfo().GetLastModified()
				}

				for _, score := range nodeCVE.GetCveBaseInfo().GetCvssMetrics() {
					cvssScores.Add(score)
				}
			}
		}

		// Try to get from cluster CVE datastore (covers K8S and Istio).
		if clusterCVE, found, err := s.clusterCVEs.Get(ctx, cveID); err == nil && found {
			foundCVE = true
			if vuln.GetSeverity() == v2.VulnerabilitySeverity_UNKNOWN_VULNERABILITY_SEVERITY {
				vuln.Severity = storagetov2.ConvertVulnerabilitySeverity(clusterCVE.GetSeverity())
			}
			if vuln.GetCvss() == 0 {
				vuln.Cvss = clusterCVE.GetCvss()
			}
			if clusterCVE.GetCveBaseInfo() != nil {
				if vuln.GetSummary() == "" {
					vuln.Summary = clusterCVE.GetCveBaseInfo().GetSummary()
				}
				if vuln.GetLink() == "" {
					vuln.Link = clusterCVE.GetCveBaseInfo().GetLink()
				}
				if vuln.GetPublishedOn() == nil {
					vuln.PublishedOn = clusterCVE.GetCveBaseInfo().GetPublishedOn()
				}
				if vuln.GetLastModified() == nil {
					vuln.LastModified = clusterCVE.GetCveBaseInfo().GetLastModified()
				}

				for _, score := range clusterCVE.GetCveBaseInfo().GetCvssMetrics() {
					cvssScores.Add(score)
				}
			}
		}

		// Only add to response if we found the CVE in at least one datastore.
		if foundCVE {
			vuln.CvssMetrics = storagetov2.ScoreVersions(cvssScores.AsSlice())
			cves[cveID] = vuln
			log.Debug("found CVE ", cveID)
		} else {
			log.Debug("CVE not found with id ", cveID)
		}
	}

	return &v2.ListCVEMetadataResponse{Cves: cves}, nil
}
