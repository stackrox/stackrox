package matcher

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/facebookincubator/nvdtools/cvefeed/nvd/schema"
	"github.com/hashicorp/go-version"
	"github.com/pkg/errors"
	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/cve/converter"
	imageDataStore "github.com/stackrox/rox/central/image/datastore"
	nsDataStore "github.com/stackrox/rox/central/namespace/datastore"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/stringutils"
)

var (
	log = logging.LoggerForModule()

	readCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Cluster, resources.Namespace, resources.Image),
		))

	gkeVersionRegex = regexp.MustCompile(`^[v|V]?[0-9]+\.[0-9]+\.[0-9]+-gke\.[0-9]+$`)
	eksVersionRegex = regexp.MustCompile(`^[v|V]?[0-9]+\.[0-9]+\.[0-9]+.*eks.*$`)
)

// CVEMatcher provides funcitonality to determine whether non-image cve is applicable to cluster
type CVEMatcher struct {
	clusters   clusterDataStore.DataStore
	namespaces nsDataStore.DataStore
	images     imageDataStore.DataStore
}

// NewCVEMatcher returns new instance of CVEMatcher
func NewCVEMatcher(clusters clusterDataStore.DataStore, namespaces nsDataStore.DataStore, images imageDataStore.DataStore) (*CVEMatcher, error) {
	return &CVEMatcher{
		clusters:   clusters,
		namespaces: namespaces,
		images:     images,
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
func (m *CVEMatcher) GetAffectedClusters(nvdCVE *schema.NVDCVEFeedJSON10DefCVEItem) ([]*storage.Cluster, error) {
	clusters, err := m.clusters.GetClusters(readCtx)
	if err != nil {
		return nil, err
	}

	filtered := make([]*storage.Cluster, 0, len(clusters))
	for _, cluster := range clusters {
		affected, err := m.IsClusterAffectedByK8sOrIstioCVE(readCtx, cluster, nvdCVE)
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
	clusterVersion := cluster.GetStatus().GetOrchestratorMetadata().GetVersion()
	for _, node := range cve.Configurations.Nodes {
		if m.matchVersions(node, clusterVersion, converter.K8s) {
			return true, nil
		}
	}
	return false, nil
}

// IsClusterAffectedByIstioCVE returns true if cluster is affected by istio cve
func (m *CVEMatcher) IsClusterAffectedByIstioCVE(ctx context.Context, cluster *storage.Cluster, entry *schema.NVDCVEFeedJSON10DefCVEItem) (bool, error) {
	ok, err := m.isIstioControlPlaneRunning(ctx)
	if err != nil {
		return false, err
	}
	if !ok {
		return false, nil
	}

	versions, err := m.getAllIstioComponentsVersionsInCluster(ctx, cluster)
	if err != nil {
		return false, err
	}
	for _, node := range entry.Configurations.Nodes {
		for _, version := range versions.AsSlice() {
			if m.matchVersions(node, version, converter.Istio) {
				return true, nil
			}
		}
	}
	return false, nil
}

func (m *CVEMatcher) isIstioControlPlaneRunning(ctx context.Context) (bool, error) {
	q := search.NewQueryBuilder().AddExactMatches(search.Namespace, "istio-system").ProtoQuery()
	res, err := m.namespaces.SearchNamespaces(ctx, q)
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
	images, err := m.images.SearchRawImages(ctx, q)
	if err != nil {
		return set, err
	}
	for _, image := range images {
		set.Add(image.GetName().GetTag())
	}
	return set, nil
}

func (m *CVEMatcher) matchVersions(node *schema.NVDCVEFeedJSON10DefNode, versionToMatch string, ct converter.CVEType) bool {
	if node.Operator != "OR" {
		log.Errorf("operator %q is not supported right now", node.Operator)
		return false
	}

	if m.IsGKEOrEKSVersion(versionToMatch) {
		versionToMatch = strings.Split(versionToMatch, "-")[0]
	}

	for _, cpeMatch := range node.CPEMatch {
		// It might be possible that the node contains non kube cpes too, so keep iterating. For example,
		// "cpe23Uri": "cpe:2.3:a:cncf:portmap:*:*:*:*:*:container_networking_interface:*:*", and
		// "cpe23Uri": "cpe:2.3:a:kubernetes:kubernetes:*:*:*:*:*:*:*:*" are in the same node
		cpeVersionAndUpdate, err := getVersionAndUpdateFromCpe(cpeMatch.Cpe23Uri, ct)
		if err != nil {
			log.Error(errors.Wrapf(err, "could not get version and update from cpe: %q", cpeMatch.Cpe23Uri))
			continue
		}

		if cpeVersionAndUpdate == "" {
			continue
		}

		// The version is N/A, treating it as a match
		if cpeVersionAndUpdate == "-:*" {
			return true
		}

		if versionToMatch == "" {
			return false
		}

		targetVersion, err := version.NewVersion(versionToMatch)
		if err != nil {
			log.Error(errors.Wrapf(err, "could not create version for cluster version: %q", versionToMatch))
			continue
		}

		// This is the case where there is just one version so check against it
		// Note that cpeVersionAndUpdate can't be "*:*" in this case, since there is no info about start and end versions
		if stringutils.AllEmpty(cpeMatch.VersionStartIncluding, cpeMatch.VersionEndIncluding, cpeMatch.VersionEndExcluding) {
			// This means this version and all prelease, build versions of this version. For example 1.6.4:*
			if strings.HasSuffix(cpeVersionAndUpdate, ":*") {
				match, err := matchBaseVersion(strings.TrimSuffix(cpeVersionAndUpdate, ":*"), versionToMatch)
				if err != nil {
					log.Error(errors.Wrapf(err, "could not compare base version %q with cluster version: %q", strings.TrimSuffix(cpeVersionAndUpdate, ":*"), versionToMatch))
					continue
				}
				if match {
					return true
				}
				continue
			}

			// Case of specific version and prerelease. Example 1.6.4:beta0
			cpeVersion := strings.Join(strings.Split(cpeVersionAndUpdate, ":"), "-")
			match, err := matchExactVersion(cpeVersion, versionToMatch)
			if err != nil {
				log.Error(errors.Wrapf(err, "could not compare exact version %q with cluster version: %q", cpeVersion, versionToMatch))
				continue
			}
			if match {
				return true
			}
		} else { // This is case where we're dealing with block of versions

			targetVersion, err := getBaseVersion(targetVersion)
			if err != nil {
				continue
			}

			var constraints []*version.Constraint

			if cpeMatch.VersionStartIncluding != "" {
				cs := getConstraints(fmt.Sprintf(">= %s", cpeMatch.VersionStartIncluding))
				constraints = append(constraints, cs...)
			}

			if cpeMatch.VersionEndIncluding != "" {
				cs := getConstraints(fmt.Sprintf("<= %s", cpeMatch.VersionEndIncluding))
				constraints = append(constraints, cs...)
			}

			if cpeMatch.VersionEndExcluding != "" {
				cs := getConstraints(fmt.Sprintf("< %s", cpeMatch.VersionEndExcluding))
				constraints = append(constraints, cs...)
			}

			val := true
			for _, c := range constraints {
				val = val && c.Check(targetVersion)
			}
			if val {
				return true
			}
		}
	}
	return false
}

func getConstraints(s string) []*version.Constraint {
	cs, err := version.NewConstraint(s)
	if err != nil {
		log.Error(err)
		return []*version.Constraint{}
	}
	return cs
}

func matchBaseVersion(version1, version2 string) (bool, error) {
	v1, err := version.NewVersion(version1)
	if err != nil {
		log.Error(err)
		return false, err
	}
	v2, err := version.NewVersion(version2)
	if err != nil {
		log.Error(err)
		return false, err
	}
	// For ex [1.6.4, 1.6.4] Or [1.6.4, 1.6.4+build1] should be matched
	if v1.Equal(v2) {
		return true, nil
	}
	// For ex [1.6.4 and 1.6.4-beta1
	v2, err = getBaseVersion(v2)
	if err != nil {
		log.Error(err)
		return false, err
	}
	return v1.Equal(v2), nil
}

func matchExactVersion(version1, version2 string) (bool, error) {
	v1, err := version.NewVersion(version1)
	if err != nil {
		return false, err
	}
	v2, err := version.NewVersion(version2)
	if err != nil {
		return false, err
	}
	return v1.Equal(v2), nil
}

func getBaseVersion(v *version.Version) (*version.Version, error) {
	prerelease := v.Prerelease()
	if prerelease == "" {
		return v, nil
	}
	versionWithoutPrerelease := strings.ReplaceAll(v.String(), "-"+prerelease, "")
	bv, err := version.NewVersion(versionWithoutPrerelease)
	if err != nil {
		return nil, err
	}
	return bv, nil
}

func getVersionAndUpdateFromCpe(cpe string, ct converter.CVEType) (string, error) {
	if ok := strings.HasPrefix(cpe, "cpe:2.3:a:"); !ok {
		return "", errors.Errorf("cpe: %q not a valid cpe23Uri format", cpe)
	}

	ss := strings.Split(cpe, ":")
	if len(ss) != 13 {
		return "", errors.Errorf("cpe: %q not a valid cpe23Uri format", cpe)
	}
	if ct != converter.K8s && ct != converter.Istio {
		return "", errors.Errorf("unkown CVE type: %d", ct)
	}
	if ct == converter.K8s && (ss[3] != "kubernetes" || ss[4] != "kubernetes") {
		return "", nil
	}
	if ct == converter.Istio && (ss[3] != "istio" || ss[4] != "istio") {
		return "", nil
	}

	return strings.Join(ss[5:7], ":"), nil
}
