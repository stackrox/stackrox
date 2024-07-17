package platformcve

import (
	"time"

	"github.com/stackrox/rox/generated/storage"
)

type platformCVECoreResponse struct {
	CVE                 string              `db:"cve"`
	CVEID               string              `db:"cve_id"`
	CVEType             storage.CVE_CVEType `db:"cve_type"`
	CVSS                float32             `db:"cvss"`
	ClusterCount        int                 `db:"cluster_id_count"`
	GenericClusters     int                 `db:"generic_cluster_count"`
	KubernetesClusters  int                 `db:"kubernetes_cluster_count"`
	OpenshiftClusters   int                 `db:"openshift_cluster_count"`
	Openshift4Clusters  int                 `db:"openshift4_cluster_count"`
	FirstDiscoveredTime *time.Time          `db:"cve_created_time"`
	FixableCount        int                 `db:"fixable_cluster_count"`
}

// GetCVE returns the CVE identifier
func (c *platformCVECoreResponse) GetCVE() string {
	return c.CVE
}

// GetCVEID returns the unique primary key ID associated with the platform CVE
func (c *platformCVECoreResponse) GetCVEID() string {
	return c.CVEID
}

// GetCVEType returns the platform CVE type
func (c *platformCVECoreResponse) GetCVEType() storage.CVE_CVEType {
	return c.CVEType
}

// GetCVSS returns the CVSS score of the platform CVE
func (c *platformCVECoreResponse) GetCVSS() float32 {
	return c.CVSS
}

// GetClusterCount returns the number of clusters affected by the platform CVE
func (c *platformCVECoreResponse) GetClusterCount() int {
	return c.ClusterCount
}

// GetClusterCountByPlatformType  returns the number of clusters of each platform type
func (c *platformCVECoreResponse) GetClusterCountByPlatformType() ClusterCountByPlatformType {
	return &clusterCountByPlatformType{
		GenericClusterCount:    c.GenericClusters,
		KubernetesClusterCount: c.KubernetesClusters,
		OpenshiftClusterCount:  c.OpenshiftClusters,
		Openshift4ClusterCount: c.Openshift4Clusters,
	}
}

// GetFixability returns true if the platform CVE is fixable in any of the affected clusters
func (c *platformCVECoreResponse) GetFixability() bool {
	return c.FixableCount > 0
}

// GetFirstDiscoveredTime returns the first time the platform CVE was discovered in the system
func (c *platformCVECoreResponse) GetFirstDiscoveredTime() *time.Time {
	return c.FirstDiscoveredTime
}

type platformCVECoreCount struct {
	CVECount int `db:"cve_id_count"`
}

type clusterCountByPlatformType struct {
	GenericClusterCount    int
	KubernetesClusterCount int
	OpenshiftClusterCount  int
	Openshift4ClusterCount int
}

func (c *clusterCountByPlatformType) GetGenericClusterCount() int {
	return c.GenericClusterCount
}

func (c *clusterCountByPlatformType) GetKubernetesClusterCount() int {
	return c.KubernetesClusterCount
}

func (c *clusterCountByPlatformType) GetOpenshiftClusterCount() int {
	return c.OpenshiftClusterCount
}

func (c *clusterCountByPlatformType) GetOpenshift4ClusterCount() int {
	return c.Openshift4ClusterCount
}

type clusterResponse struct {
	ClusterID string `db:"cluster_id"`

	// Following are supported sort options.
	ClusterName       string              `db:"cluster"`
	ClusterType       storage.ClusterType `db:"cluster_platform_type"`
	KubernetesVersion string              `db:"cluster_kubernetes_version"`
}

type cveCountByTypeResponse struct {
	KubernetesCVECount int `db:"k8s_cve_count"`
	OpenshiftCVECount  int `db:"openshift_cve_count"`
	IstioCVECount      int `db:"istio_cve_count"`
}

func (c *cveCountByTypeResponse) GetKubernetesCVECount() int {
	return c.KubernetesCVECount
}

func (c *cveCountByTypeResponse) GetOpenshiftCVECount() int {
	return c.OpenshiftCVECount
}

func (c *cveCountByTypeResponse) GetIstioCVECount() int {
	return c.IstioCVECount
}

type cveCountByFixabilityResponse struct {
	CVECount     int `db:"cve_id_count"`
	FixableCount int `db:"fixable_cve_id_count"`
}

func (c *cveCountByFixabilityResponse) GetTotal() int {
	return c.CVECount
}

func (c *cveCountByFixabilityResponse) GetFixable() int {
	return c.FixableCount
}
