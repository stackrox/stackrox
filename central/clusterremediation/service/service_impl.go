package service

import (
	"context"
	"time"

	"github.com/google/go-containerregistry/pkg/authn"
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
	imageTypes "github.com/stackrox/rox/pkg/images/types"
	imageUtils "github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/operatorbundle"
	"github.com/stackrox/rox/pkg/operatorbundle/ocprelease"
	"github.com/stackrox/rox/pkg/registries"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/uuid"
	"google.golang.org/grpc"
	"k8s.io/client-go/dynamic"
)

// releaseRepo is the OpenShift release registry repository; used only to resolve pull credentials
// from a matching registry integration (the actual release images come from the ClusterVersion CR).
const releaseRepo = "quay.io/openshift-release-dev/ocp-release"

// jobTimeout bounds a report generation (it scans many candidate images).
const jobTimeout = 20 * time.Minute

var authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
	user.With(permissions.View(resources.Image)): {
		v1.ClusterRemediationService_CreateClusterRemediationReport_FullMethodName,
		v1.ClusterRemediationService_GetClusterRemediationReport_FullMethodName,
	},
})

type jobState struct {
	status v1.ClusterRemediationJob_Status
	report *v1.ClusterRemediationReport
	errMsg string
}

type serviceImpl struct {
	v1.UnimplementedClusterRemediationServiceServer

	images *datasource.CentralImages
	dyn    dynamic.Interface
	regSet registries.Set

	mu   *sync.Mutex
	jobs map[string]*jobState
}

func (s *serviceImpl) RegisterServiceServer(server *grpc.Server) {
	v1.RegisterClusterRemediationServiceServer(server, s)
}

func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterClusterRemediationServiceHandler(ctx, mux, conn)
}

func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

// CreateClusterRemediationReport starts report generation in the background and returns a RUNNING job.
func (s *serviceImpl) CreateClusterRemediationReport(_ context.Context, req *v1.ClusterRemediationRequest) (*v1.ClusterRemediationJob, error) {
	if s.dyn == nil {
		return nil, errors.Wrap(errox.NotImplemented, "in-cluster OpenShift release access is not available on this Central")
	}
	id := uuid.NewV4().String()
	s.mu.Lock()
	s.jobs[id] = &jobState{status: v1.ClusterRemediationJob_RUNNING}
	s.mu.Unlock()

	go s.run(id, req.GetRunningOnly())

	return &v1.ClusterRemediationJob{Id: id, Status: v1.ClusterRemediationJob_RUNNING}, nil
}

// GetClusterRemediationReport returns the current state (and result, when COMPLETE) of a job.
func (s *serviceImpl) GetClusterRemediationReport(_ context.Context, req *v1.GetClusterRemediationJobRequest) (*v1.ClusterRemediationJob, error) {
	s.mu.Lock()
	state, ok := s.jobs[req.GetJobId()]
	s.mu.Unlock()
	if !ok {
		return nil, errors.Wrapf(errox.NotFound, "no report job %q", req.GetJobId())
	}
	return &v1.ClusterRemediationJob{
		Id:     req.GetJobId(),
		Status: state.status,
		Error:  state.errMsg,
		Report: state.report,
	}, nil
}

func (s *serviceImpl) finish(id string, report *v1.ClusterRemediationReport, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	state := s.jobs[id]
	if state == nil {
		return
	}
	if err != nil {
		state.status = v1.ClusterRemediationJob_FAILED
		state.errMsg = err.Error()
		return
	}
	state.status = v1.ClusterRemediationJob_COMPLETE
	state.report = report
}

func (s *serviceImpl) run(id string, runningOnly bool) {
	ctx, cancel := context.WithTimeout(sac.WithAllAccess(context.Background()), jobTimeout)
	defer cancel()

	catalog := ocprelease.NewClient(s.dyn, s.releaseAuth())
	opts := []operatorbundle.AdvisorOption{}
	if runningOnly {
		opts = append(opts, operatorbundle.WithRunningOnly(s.images))
	}
	advisor := operatorbundle.NewAdvisor(catalog, s.images, s.images, opts...)

	// The triggering digest is ignored by the release catalog client (the current release is read
	// from the ClusterVersion CR); pass the synthetic package name as the entry point.
	reports, _, err := advisor.Advise(ctx, []string{ocprelease.Package})
	if err != nil {
		s.finish(id, nil, errors.Wrap(err, "generating cluster remediation report"))
		return
	}
	if len(reports) == 0 {
		s.finish(id, nil, errors.New("no current OpenShift release resolved"))
		return
	}
	s.finish(id, toReport(reports[0]), nil)
}

// releaseAuth resolves pull credentials for the OpenShift release registry from a matching registry
// integration (same path the enricher uses). Falls back to anonymous when none is configured.
func (s *serviceImpl) releaseAuth() authn.Authenticator {
	ci, err := imageUtils.GenerateImageFromString(releaseRepo)
	if err != nil {
		return authn.Anonymous
	}
	cfg := s.regSet.GetRegistryMetadataByImage(imageTypes.ToImage(ci))
	if cfg == nil || cfg.Username == "" {
		return authn.Anonymous
	}
	return authn.FromConfig(authn.AuthConfig{Username: cfg.Username, Password: cfg.Password})
}

func toReport(r operatorbundle.BundleDiffReport) *v1.ClusterRemediationReport {
	report := &v1.ClusterRemediationReport{
		CurrentVersion: r.InstalledBundle.Version,
		NoUpdateReason: r.NoUpdateReason,
	}
	if r.UpdateCandidate != nil {
		report.TargetVersion = r.UpdateCandidate.Version
	}
	for _, d := range r.ImageDiffs {
		report.Images = append(report.Images, &v1.ImageRemediation{
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
	return report
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
