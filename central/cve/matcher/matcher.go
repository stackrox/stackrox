package matcher

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/Masterminds/semver"
	"github.com/facebookincubator/nvdtools/cvefeed/nvd/schema"
	"github.com/pkg/errors"
	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/cve/converter/utils"
	imageDataStore "github.com/stackrox/rox/central/image/datastore"
	imageV2DataStore "github.com/stackrox/rox/central/imagev2/datastore"
	nsDataStore "github.com/stackrox/rox/central/namespace/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/stringutils"
)

var (
	log = logging.LoggerForModule()

	gkeVersionRegex = regexp.MustCompile(`^[vV]?[0-9]+\.[0-9]+\.[0-9]+-gke\.[0-9]+$`)
	eksVersionRegex = regexp.MustCompile(`^[vV]?[0-9]+\.[0-9]+\.[0-9]+.*eks.*$`)
)

// CVEMatcher provides functionality to determine whether non-image cve is applicable to cluster
type CVEMatcher struct {
	clusters   clusterDataStore.DataStore
	namespaces nsDataStore.DataStore
	images     imageDataStore.DataStore
	imagesV2   imageV2DataStore.DataStore
}

// NewCVEMatcher returns new instance of CVEMatcher
func NewCVEMatcher(clusters clusterDataStore.DataStore, namespaces nsDataStore.DataStore, images imageDataStore.DataStore, imagesV2 imageV2DataStore.DataStore) (*CVEMatcher, error) {
	return &CVEMatcher{
		clusters:   clusters,
		namespaces: namespaces,
		images:     images,
		imagesV2:   imagesV2,
	}, nil
}

// IsClusterCVEFixable returns if the true if cluster cve is fixable
func IsClusterCVEFixable(cve *schema.NVDCVEFeedJSON10DefCVEItem) bool {
	for _, node := range cve.Configurations.Nodes {
		for _, cpeMatch := range node.CPEMatch {
			if cpeMatch.VersionEndExcluding != "" {
				return true
			}
		}
	}
	return false
}

// IsGKEOrEKSVersion determines if given version string is GKE or EKS
func (m *CVEMatcher) IsGKEOrEKSVersion(version string) bool {
	return m.IsGKEVersion(version) || m.IsEKSVersion(version)
}

// IsGKEVersion determines if given version string is GKE
func (m *CVEMatcher) IsGKEVersion(version string) bool {
	return gkeVersionRegex.MatchString(version)
}

// IsEKSVersion determines if given version is EKS
func (m *CVEMatcher) IsEKSVersion(version string) bool {
	return eksVersionRegex.MatchString(version)
}

// GetAffectedClusters returns the clusters affected by k8s and istio cves
func (m *CVEMatcher) GetAffectedClusters(ctx context.Context, nvdCVE *schema.NVDCVEFeedJSON10DefCVEItem) ([]*storage.Cluster, error) {
	clusters, err := m.clusters.GetClusters(ctx)
	if err != nil {
		return nil, err
	}

	filtered := make([]*storage.Cluster, 0, len(clusters))
	for _, cluster := range clusters {
		affected, err := m.IsClusterAffectedByK8sOrIstioCVE(ctx, cluster, nvdCVE)
		if err != nil {
			return nil, err
		}

		if !affected {
			continue
		}
		filtered = append(filtered, cluster)
	}
	return filtered, nil
}

// IsClusterAffectedByK8sOrIstioCVE returns true if cluster is affected by k8s and istio cve
func (m *CVEMatcher) IsClusterAffectedByK8sOrIstioCVE(ctx context.Context, cluster *storage.Cluster, cve *schema.NVDCVEFeedJSON10DefCVEItem) (bool, error) {
	affected1, err := m.IsClusterAffectedByIstioCVE(ctx, cluster, cve)
	if err != nil {
		return false, err
	}
	affected2, err := m.IsClusterAffectedByK8sCVE(ctx, cluster, cve)
	if err != nil {
		return false, err
	}
	return affected1 || affected2, nil
}

// IsClusterAffectedByK8sCVE returns true if cluster is affected by k8s cve
func (m *CVEMatcher) IsClusterAffectedByK8sCVE(_ context.Context, cluster *storage.Cluster, cve *schema.NVDCVEFeedJSON10DefCVEItem) (bool, error) {
	if cve.Configurations == nil {
		return false, nil
	}
	clusterVersion := cluster.GetStatus().GetOrchestratorMetadata().GetVersion()
	for _, node := range cve.Configurations.Nodes {
		matched, err := m.MatchVersions(node, clusterVersion, utils.K8s)
		// If we could determine CVE impact from one of cpe string, we skip logging error
		if matched {
			return true, nil
		}
		if err != nil {
			log.Error(errors.Wrapf(err, "errors occurred determining impact of k8s cve %s", cve.CVE.CVEDataMeta.ID))
		}
	}
	return false, nil
}

// IsClusterAffectedByIstioCVE returns true if cluster is affected by istio cve
func (m *CVEMatcher) IsClusterAffectedByIstioCVE(ctx context.Context, cluster *storage.Cluster, cve *schema.NVDCVEFeedJSON10DefCVEItem) (bool, error) {
	versions, err := m.GetValidIstioVersions(ctx, cluster)
	if err != nil {
		return false, err
	}
	if len(versions) == 0 {
		return false, nil
	}
	for _, node := range cve.Configurations.Nodes {
		for version := range versions {
			matched, err := m.MatchVersions(node, version, utils.Istio)
			// If we could determine CVE impact from one of cpe string, we skip logging error
			if matched {
				return true, nil
			}
			if err != nil {
				log.Error(errors.Wrapf(err, "errors occurred determining impact of istio cve %s", cve.CVE.CVEDataMeta.ID))
			}
		}
	}
	return false, nil
}

// GetValidIstioVersions returns all running Istio versions in the given cluster, if there are any, and nil otherwise
func (m *CVEMatcher) GetValidIstioVersions(ctx context.Context, cluster *storage.Cluster) (set.StringSet, error) {
	ok, err := m.isIstioControlPlaneRunning(ctx)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}

	versions, err := m.getAllIstioComponentsVersionsInCluster(ctx, cluster)
	if err != nil {
		return nil, err
	}

	return versions, nil
}

func (m *CVEMatcher) isIstioControlPlaneRunning(ctx context.Context) (bool, error) {
	q := search.NewQueryBuilder().AddExactMatches(search.Namespace, "istio-system").ProtoQuery()
	res, err := m.namespaces.Search(ctx, q)
	if err != nil {
		return false, err
	}
	return len(res) > 0, nil
}

func (m *CVEMatcher) getAllIstioComponentsVersionsInCluster(ctx context.Context, cluster *storage.Cluster) (set.StringSet, error) {
	set := set.StringSet{}
	q := search.NewQueryBuilder().
		AddExactMatches(search.ClusterID, cluster.GetId()).
		AddExactMatches(search.ImageRegistry, "docker.io").
		AddStrings(search.ImageRemote, "istio").
		ProtoQuery()
	if features.FlattenImageData.Enabled() {
		images, err := m.imagesV2.SearchRawImages(ctx, q)
		if err != nil {
			return set, err
		}
		for _, image := range images {
			set.Add(image.GetName().GetTag())
		}
	} else {
		images, err := m.images.SearchRawImages(ctx, q)
		if err != nil {
			return set, err
		}
		for _, image := range images {
			set.Add(image.GetName().GetTag())
		}
	}
	return set, nil
}

// MatchVersions returns if versionToMatch is affected by cve according to its config node.
func (m *CVEMatcher) MatchVersions(node *schema.NVDCVEFeedJSON10DefNode, versionToMatch string, ct utils.CVEType) (bool, error) {
	if node.Operator != "OR" {
		return false, nil
	}

	if m.IsGKEOrEKSVersion(versionToMatch) {
		versionToMatch = strings.Split(versionToMatch, "-")[0]
	}

	var errList errorhelpers.ErrorList
	for _, cpeMatch := range node.CPEMatch {
		// It might be possible that the node contains non kube cpes too, so keep iterating. For example,
		// "cpe23Uri": "cpe:2.3:a:cncf:portmap:*:*:*:*:*:container_networking_interface:*:*", and
		// "cpe23Uri": "cpe:2.3:a:kubernetes:kubernetes:*:*:*:*:*:*:*:*" are in the same node
		cpeVersionAndUpdate := getVersionAndUpdateFromCpe(cpeMatch.Cpe23Uri, ct)
		if cpeVersionAndUpdate == "" {
			continue
		}

		// The version is N/A, treating it as a match
		if cpeVersionAndUpdate == "-:*" {
			return true, errList.ToError()
		}

		if versionToMatch == "" {
			return false, errList.ToError()
		}

		targetVersion, err := semver.NewVersion(versionToMatch)
		if err != nil {
			// Fail-open: if we can't parse the cluster version, assume it's affected.
			// For a security product, "I can't determine safety" should mean "assume vulnerable."
			log.Error(errors.Wrapf(err, "could not parse cluster version %q — assuming affected", versionToMatch))
			return true, errList.ToError()
		}

		// This is the case where there is just one version so check against it.
		// Note that cpeVersionAndUpdate can't be "*:*" in this case, since there is no info about start and end versions.
		if stringutils.AllEmpty(cpeMatch.VersionStartIncluding, cpeMatch.VersionEndIncluding, cpeMatch.VersionEndExcluding) {
			// This means this version and all prerelease, build versions of this version. For example 1.6.4:*
			if before, ok := strings.CutSuffix(cpeVersionAndUpdate, ":*"); ok {
				cpeVer, err := semver.NewVersion(before)
				if err != nil {
					errList.AddError(errors.Wrapf(err, "could not compare base version %q with cluster version: %q", before, versionToMatch))
					continue
				}
				// Match if versions are equal (handles build metadata) or if
				// base versions are equal (handles prerelease variants).
				if cpeVer.Equal(targetVersion) || cpeVer.Equal(getBaseVersion(targetVersion)) {
					return true, errList.ToError()
				}
				continue
			}

			// Case of specific version and prerelease. Example 1.6.4:beta0
			cpeVersion := strings.Join(strings.Split(cpeVersionAndUpdate, ":"), "-")
			cpeVer, err := semver.NewVersion(cpeVersion)
			if err != nil {
				errList.AddError(errors.Wrapf(err, "could not compare exact version %q with cluster version: %q", cpeVersion, versionToMatch))
				continue
			}
			if cpeVer.Equal(targetVersion) {
				return true, errList.ToError()
			}
		} else {
			// This is case where we're dealing with block of versions.
			// Use base version (no prerelease/metadata) for range comparison.
			baseVersion := getBaseVersion(targetVersion)

			var parts []string
			if cpeMatch.VersionStartIncluding != "" {
				parts = append(parts, fmt.Sprintf(">= %s", cpeMatch.VersionStartIncluding))
			}

			if cpeMatch.VersionEndIncluding != "" {
				parts = append(parts, fmt.Sprintf("<= %s", cpeMatch.VersionEndIncluding))
			}

			if cpeMatch.VersionEndExcluding != "" {
				parts = append(parts, fmt.Sprintf("< %s", cpeMatch.VersionEndExcluding))
			}

			// Unlike the prior implementation using hashicorp/go-version, where
			// individually-parsed constraints were AND'd in a loop -- meaning
			// if all parses failed the empty slice defaulted to true (false
			// positive) -- this parses the full constraint string at once and
			// skips on error, correctly returning no-match.
			constraint, err := semver.NewConstraint(strings.Join(parts, ", "))
			if err != nil {
				log.Error(err)
				continue
			}
			if constraint.Check(baseVersion) {
				return true, errList.ToError()
			}
		}
	}
	return false, errList.ToError()
}

// getBaseVersion strips prerelease and metadata, returning only major.minor.patch.
// The error from NewVersion is impossible with three non-negative integers.
func getBaseVersion(v *semver.Version) *semver.Version {
	if v.Prerelease() == "" {
		return v
	}
	base, _ := semver.NewVersion(fmt.Sprintf("%d.%d.%d", v.Major(), v.Minor(), v.Patch()))
	return base
}

func getVersionAndUpdateFromCpe(cpe string, ct utils.CVEType) string {
	if ok := strings.HasPrefix(cpe, "cpe:2.3:a:"); !ok {
		return ""
	}

	ss := strings.Split(cpe, ":")
	if len(ss) != 13 {
		return ""
	}
	if ct != utils.K8s && ct != utils.Istio && ct != utils.OpenShift {
		return ""
	}
	if ct == utils.K8s && (ss[3] != "kubernetes" || ss[4] != "kubernetes") {
		return ""
	}
	if ct == utils.Istio && (ss[3] != "istio" || ss[4] != "istio") {
		return ""
	}
	if ct == utils.OpenShift && (ss[3] != "openshift" || ss[4] != "openshift") {
		return ""
	}

	return strings.Join(ss[5:7], ":")
}
