package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/operatorremediation/datasource"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/operatorbundle"
	"github.com/stackrox/rox/pkg/operatorbundle/olm"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/utils"
	"google.golang.org/grpc"
	"k8s.io/client-go/dynamic"
)

var authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
	user.With(permissions.View(resources.Image)): {
		v1.OperatorRemediationService_GetImageRemediation_FullMethodName,
	},
})

type serviceImpl struct {
	v1.UnimplementedOperatorRemediationServiceServer

	images *datasource.CentralImages
	dyn    dynamic.Interface
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(server *grpc.Server) {
	v1.RegisterOperatorRemediationServiceServer(server, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterOperatorRemediationServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

// GetImageRemediation resolves the operator bundle shipping the requested image and returns the
// per-image CVE diff against the recommended update, computed live from in-cluster OLM data and
// Central scan results.
func (s *serviceImpl) GetImageRemediation(ctx context.Context, request *v1.GetImageRemediationRequest) (*v1.ImageRemediationResponse, error) {
	imageID := request.GetImageId()
	if imageID == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "image_id is required")
	}
	if s.dyn == nil {
		return nil, errors.Wrap(errox.NotImplemented, "in-cluster OLM access is not available on this Central")
	}

	// The advisor crosses images, deployments and OLM resources; use an elevated context for the
	// internal computation (the caller is already authorized via AuthFuncOverride).
	elevatedCtx := sac.WithAllAccess(ctx)

	digest, err := s.images.ResolveDigest(elevatedCtx, imageID)
	if err != nil {
		return nil, err
	}
	if digest == "" {
		return nil, errors.Wrapf(errox.NotFound, "image %q not found in Central", imageID)
	}

	// Resolve the installed bundle (OLM, k8s only) to learn the package, then dial its CatalogSource.
	installed, err := olm.NewClient(s.dyn, nil).FindInstalledBundle(elevatedCtx, digest)
	if err != nil {
		return nil, errors.Wrap(err, "resolving installed operator bundle")
	}
	if installed == nil {
		return nil, errors.Wrapf(errox.NotFound, "image %q is not shipped by an installed operator bundle", imageID)
	}

	addr, err := olm.ResolveCatalogGRPCAddress(elevatedCtx, s.dyn, installed.Package, installed.CSVName)
	if err != nil {
		return nil, errors.Wrap(err, "resolving operator catalog address")
	}
	conn, reg, err := olm.DialCatalogRegistry(addr)
	if err != nil {
		return nil, err
	}
	defer utils.IgnoreError(conn.Close)

	catalog := olm.NewClient(s.dyn, reg)
	opts := []operatorbundle.AdvisorOption{}
	if request.GetRunningOnly() {
		opts = append(opts, operatorbundle.WithRunningOnly(s.images))
	}
	advisor := operatorbundle.NewAdvisor(catalog, s.images, s.images, opts...)

	reports, _, err := advisor.Advise(elevatedCtx, []string{digest})
	if err != nil {
		return nil, errors.Wrap(err, "computing operator image remediation")
	}
	if len(reports) == 0 {
		return nil, errors.Wrapf(errox.NotFound, "no operator bundle resolved for image %q", imageID)
	}
	return toResponse(reports[0]), nil
}

func toResponse(r operatorbundle.BundleDiffReport) *v1.ImageRemediationResponse {
	resp := &v1.ImageRemediationResponse{
		OperatorPackage:  r.Package,
		InstalledVersion: r.InstalledBundle.Version,
		NoUpdateReason:   r.NoUpdateReason,
	}
	if r.UpdateCandidate != nil {
		resp.UpdateVersion = r.UpdateCandidate.Version
	}
	for _, d := range r.ImageDiffs {
		resp.Images = append(resp.Images, &v1.ImageRemediation{
			Repository:     d.Repository,
			Name:           d.Name,
			InstalledImage: imageRef(d.Repository, d.InstalledDigest),
			CandidateImage: imageRef(d.Repository, d.CandidateDigest),
			Status:         string(d.Status),
			Current:        countBySeverity(append(append([]operatorbundle.CVE{}, d.Fixed...), d.StillActive...)),
			Fixed:          countBySeverity(d.Fixed),
			NewCves:        countBySeverity(d.New),
		})
	}
	return resp
}

func imageRef(repository, digest string) string {
	if digest == "" {
		return ""
	}
	return repository + "@" + digest
}

func countBySeverity(cves []operatorbundle.CVE) *v1.SeverityCounts {
	sc := &v1.SeverityCounts{}
	for _, c := range cves {
		switch c.Severity {
		case storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY:
			sc.Critical++
		case storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY:
			sc.Important++
		case storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY:
			sc.Moderate++
		case storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY:
			sc.Low++
		default:
			sc.Unknown++
		}
	}
	return sc
}
