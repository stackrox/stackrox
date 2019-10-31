package resolvers

import (
	"testing"

	"github.com/stackrox/k8s-istio-cve-pusher/nvd"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/assert"
)

func TestMapImagesToVulnerabilityResolvers(t *testing.T) {
	fakeRoot := &Resolver{}
	images := testImages()

	query := &v1.Query{}
	vulnerabilityResolvers, err := mapImagesToVulnerabilityResolvers(fakeRoot, images, query)
	assert.NoError(t, err)
	assert.Len(t, vulnerabilityResolvers, 5)

	query = search.NewQueryBuilder().AddExactMatches(search.FixedBy, "1.1").ProtoQuery()
	vulnerabilityResolvers, err = mapImagesToVulnerabilityResolvers(fakeRoot, images, query)
	assert.NoError(t, err)
	assert.Len(t, vulnerabilityResolvers, 1)

	query = search.NewQueryBuilder().AddExactMatches(search.CVE, "cve-2019-1", "cve-2019-2", "cve-2019-3").ProtoQuery()
	vulnerabilityResolvers, err = mapImagesToVulnerabilityResolvers(fakeRoot, images, query)
	assert.NoError(t, err)
	assert.Len(t, vulnerabilityResolvers, 2)
}

func TestIfSpecificVersionCVEAffectsCluster(t *testing.T) {
	cluster := &storage.Cluster{
		Id:   "test_cluster_id1",
		Name: "cluster1",
		Status: &storage.ClusterStatus{
			OrchestratorMetadata: &storage.OrchestratorMetadata{
				Version: "v1.15.3",
			},
		},
	}

	cve1 := nvd.CVEEntry{
		CVE: nvd.CVE{
			Metadata: nvd.CVEMetadata{
				CVEID: "CVE-1",
			},
		},
		Configurations: nvd.Configuration{
			Nodes: []nvd.Node{
				{
					Operator: "OR",
					CPEMatch: []nvd.CPEMatch{
						{
							Vulnerable: true,
							CPE23Uri:   "cpe:2.3:a:kubernetes:kubernetes:1.15.3:*:*:*:*:*:*:*",
						},
					},
				},
			},
		},
	}

	cve2 := nvd.CVEEntry{
		CVE: nvd.CVE{
			Metadata: nvd.CVEMetadata{
				CVEID: "CVE-2",
			},
		},
		Configurations: nvd.Configuration{
			Nodes: []nvd.Node{
				{
					Operator: "OR",
					CPEMatch: []nvd.CPEMatch{
						{
							Vulnerable: true,
							CPE23Uri:   "cpe:2.3:a:kubernetes:kubernetes:1.14.3:*:*:*:*:*:*:*",
						},
					},
				},
			},
		},
	}

	cve3 := nvd.CVEEntry{
		CVE: nvd.CVE{
			Metadata: nvd.CVEMetadata{
				CVEID: "CVE-3",
			},
		},
		Configurations: nvd.Configuration{
			Nodes: []nvd.Node{
				{
					Operator: "OR",
					CPEMatch: []nvd.CPEMatch{
						{
							Vulnerable: true,
							CPE23Uri:   "cpe:2.3:a:kubernetes:kubernetes:1.15.3:alpha1:*:*:*:*:*:*",
						},
					},
				},
			},
		},
	}

	ret := isClusterAffectedByCVE(cluster, &cve1)
	assert.Equal(t, true, ret)
	ret = isClusterAffectedByCVE(cluster, &cve2)
	assert.Equal(t, false, ret)
	ret = isClusterAffectedByCVE(cluster, &cve3)
	assert.Equal(t, false, ret)

	cluster.Status.OrchestratorMetadata.Version = "v1.15.3+build1"
	ret = isClusterAffectedByCVE(cluster, &cve1)
	assert.Equal(t, true, ret)
	ret = isClusterAffectedByCVE(cluster, &cve2)
	assert.Equal(t, false, ret)
	ret = isClusterAffectedByCVE(cluster, &cve3)
	assert.Equal(t, false, ret)

	cluster.Status.OrchestratorMetadata.Version = "v1.15.3-alpha1"
	ret = isClusterAffectedByCVE(cluster, &cve1)
	assert.Equal(t, true, ret)
	ret = isClusterAffectedByCVE(cluster, &cve2)
	assert.Equal(t, false, ret)
	ret = isClusterAffectedByCVE(cluster, &cve3)
	assert.Equal(t, true, ret)

	cluster.Status.OrchestratorMetadata.Version = "v1.15.3-alpha1+build1"
	ret = isClusterAffectedByCVE(cluster, &cve1)
	assert.Equal(t, true, ret)
	ret = isClusterAffectedByCVE(cluster, &cve2)
	assert.Equal(t, false, ret)
	ret = isClusterAffectedByCVE(cluster, &cve3)
	assert.Equal(t, true, ret)
}

func TestMultipleVersionCVEAffectsCluster(t *testing.T) {
	cluster := &storage.Cluster{
		Id:   "test_cluster_id1",
		Name: "cluster1",
		Status: &storage.ClusterStatus{
			OrchestratorMetadata: &storage.OrchestratorMetadata{
				Version: "v1.15.3",
			},
		},
	}

	cve1 := nvd.CVEEntry{
		CVE: nvd.CVE{
			Metadata: nvd.CVEMetadata{
				CVEID: "CVE-1",
			},
		},
		Configurations: nvd.Configuration{
			Nodes: []nvd.Node{
				{
					Operator: "OR",
					CPEMatch: []nvd.CPEMatch{
						{
							Vulnerable:            true,
							CPE23Uri:              "cpe:2.3:a:kubernetes:kubernetes:*:*:*:*:*:*:*:*",
							VersionStartIncluding: "1.10.1",
							VersionEndExcluding:   "1.10.9",
						},
						{
							Vulnerable:            true,
							CPE23Uri:              "cpe:2.3:a:kubernetes:kubernetes:*:*:*:*:*:*:*:*",
							VersionStartIncluding: "1.11.1",
							VersionEndExcluding:   "1.11.9",
						},
					},
				},
			},
		},
	}

	cve2 := nvd.CVEEntry{
		CVE: nvd.CVE{
			Metadata: nvd.CVEMetadata{
				CVEID: "CVE-2",
			},
		},
		Configurations: nvd.Configuration{
			Nodes: []nvd.Node{
				{
					Operator: "OR",
					CPEMatch: []nvd.CPEMatch{
						{
							Vulnerable:            true,
							CPE23Uri:              "cpe:2.3:a:kubernetes:kubernetes:*:*:*:*:*:*:*:*",
							VersionStartIncluding: "1.14.1",
							VersionEndExcluding:   "1.14.9",
						},
						{
							Vulnerable:            true,
							CPE23Uri:              "cpe:2.3:a:kubernetes:kubernetes:*:*:*:*:*:*:*:*",
							VersionStartIncluding: "1.15.1",
							VersionEndExcluding:   "1.15.9",
						},
					},
				},
			},
		},
	}

	cve3 := nvd.CVEEntry{
		CVE: nvd.CVE{
			Metadata: nvd.CVEMetadata{
				CVEID: "CVE-2",
			},
		},
		Configurations: nvd.Configuration{
			Nodes: []nvd.Node{
				{
					Operator: "OR",
					CPEMatch: []nvd.CPEMatch{
						{
							Vulnerable:            true,
							CPE23Uri:              "cpe:2.3:a:kubernetes:kubernetes:*:*:*:*:*:*:*:*",
							VersionStartIncluding: "1.16.5",
							VersionEndIncluding:   "1.16.9",
						},
						{
							Vulnerable:            true,
							CPE23Uri:              "cpe:2.3:a:kubernetes:kubernetes:*:*:*:*:*:*:*:*",
							VersionStartIncluding: "1.17.1",
							VersionEndIncluding:   "1.17.9",
						},
					},
				},
			},
		},
	}

	ret := isClusterAffectedByCVE(cluster, &cve1)
	assert.Equal(t, false, ret)

	ret = isClusterAffectedByCVE(cluster, &cve2)
	assert.Equal(t, true, ret)

	cluster.Status.OrchestratorMetadata.Version = "v1.15.1-beta1"
	ret = isClusterAffectedByCVE(cluster, &cve1)
	assert.Equal(t, false, ret)

	ret = isClusterAffectedByCVE(cluster, &cve2)
	assert.Equal(t, false, ret)

	cluster.Status.OrchestratorMetadata.Version = "v1.15.9"
	ret = isClusterAffectedByCVE(cluster, &cve1)
	assert.Equal(t, false, ret)

	ret = isClusterAffectedByCVE(cluster, &cve2)
	assert.Equal(t, false, ret)

	cluster.Status.OrchestratorMetadata.Version = "v1.15.1"
	ret = isClusterAffectedByCVE(cluster, &cve1)
	assert.Equal(t, false, ret)

	ret = isClusterAffectedByCVE(cluster, &cve2)
	assert.Equal(t, true, ret)

	cluster.Status.OrchestratorMetadata.Version = "v1.16.4"
	ret = isClusterAffectedByCVE(cluster, &cve3)
	assert.Equal(t, false, ret)

	cluster.Status.OrchestratorMetadata.Version = "v1.17.4"
	ret = isClusterAffectedByCVE(cluster, &cve3)
	assert.Equal(t, true, ret)
}

func TestSingleAndMultipleVersionCVEAffectsCluster(t *testing.T) {
	cluster := &storage.Cluster{
		Id:   "test_cluster_id1",
		Name: "cluster1",
		Status: &storage.ClusterStatus{
			OrchestratorMetadata: &storage.OrchestratorMetadata{
				Version: "v1.10.6",
			},
		},
	}
	cve := nvd.CVEEntry{
		CVE: nvd.CVE{
			Metadata: nvd.CVEMetadata{
				CVEID: "CVE-1",
			},
		},
		Configurations: nvd.Configuration{
			Nodes: []nvd.Node{
				{
					Operator: "OR",
					CPEMatch: []nvd.CPEMatch{
						{
							Vulnerable:            true,
							CPE23Uri:              "cpe:2.3:a:kubernetes:kubernetes:*:*:*:*:*:*:*:*",
							VersionStartIncluding: "1.10.1",
							VersionEndExcluding:   "1.10.9",
						},
						{
							Vulnerable:            true,
							CPE23Uri:              "cpe:2.3:a:kubernetes:kubernetes:*:*:*:*:*:*:*:*",
							VersionStartIncluding: "1.11.1",
							VersionEndExcluding:   "1.11.9",
						},
						{
							Vulnerable: true,
							CPE23Uri:   "cpe:2.3:a:kubernetes:kubernetes:1.10.3:alpha1:*:*:*:*:*:*",
						},
						{
							Vulnerable: true,
							CPE23Uri:   "cpe:2.3:a:kubernetes:kubernetes:1.10.3:beta1:*:*:*:*:*:*",
						},
					},
				},
			},
		},
	}

	ret := isClusterAffectedByCVE(cluster, &cve)
	assert.Equal(t, true, ret)

	cluster.Status.OrchestratorMetadata.Version = "v1.10.3-alpha1"
	ret = isClusterAffectedByCVE(cluster, &cve)
	assert.Equal(t, true, ret)

	cluster.Status.OrchestratorMetadata.Version = "v1.10.3-beta1"
	ret = isClusterAffectedByCVE(cluster, &cve)
	assert.Equal(t, true, ret)

	cluster.Status.OrchestratorMetadata.Version = "v1.10.3-rc1"
	ret = isClusterAffectedByCVE(cluster, &cve)
	assert.Equal(t, false, ret)
}

func TestCountCVEsAffectsCluster(t *testing.T) {
	cluster := &storage.Cluster{
		Id:   "test_cluster_id1",
		Name: "cluster1",
		Status: &storage.ClusterStatus{
			OrchestratorMetadata: &storage.OrchestratorMetadata{
				Version: "v1.10.6",
			},
		},
	}
	cves := []nvd.CVEEntry{
		{
			CVE: nvd.CVE{
				Metadata: nvd.CVEMetadata{
					CVEID: "CVE-1",
				},
			},
			Configurations: nvd.Configuration{
				Nodes: []nvd.Node{
					{
						Operator: "OR",
						CPEMatch: []nvd.CPEMatch{
							{
								Vulnerable:            true,
								CPE23Uri:              "cpe:2.3:a:kubernetes:kubernetes:*:*:*:*:*:*:*:*",
								VersionStartIncluding: "1.10.1",
								VersionEndExcluding:   "1.10.9",
							},
						},
					},
				},
			},
		},
		{
			CVE: nvd.CVE{
				Metadata: nvd.CVEMetadata{
					CVEID: "CVE-2",
				},
			},
			Configurations: nvd.Configuration{
				Nodes: []nvd.Node{
					{
						Operator: "OR",
						CPEMatch: []nvd.CPEMatch{
							{
								Vulnerable: true,
								CPE23Uri:   "cpe:2.3:a:kubernetes:kubernetes:1.10.6:*:*:*:*:*:*:*",
							},
						},
					},
				},
			},
		},
		{
			CVE: nvd.CVE{
				Metadata: nvd.CVEMetadata{
					CVEID: "CVE-3",
				},
			},
			Configurations: nvd.Configuration{
				Nodes: []nvd.Node{
					{
						Operator: "OR",
						CPEMatch: []nvd.CPEMatch{
							{
								Vulnerable:            true,
								CPE23Uri:              "cpe:2.3:a:kubernetes:kubernetes:*:*:*:*:*:*:*:*",
								VersionStartIncluding: "1.10.3",
								VersionEndIncluding:   "1.10.7",
							},
						},
					},
				},
			},
		},
		{
			CVE: nvd.CVE{
				Metadata: nvd.CVEMetadata{
					CVEID: "CVE-4",
				},
			},
			Configurations: nvd.Configuration{
				Nodes: []nvd.Node{
					{
						Operator: "OR",
						CPEMatch: []nvd.CPEMatch{
							{
								Vulnerable:            true,
								CPE23Uri:              "cpe:2.3:a:kubernetes:kubernetes:*:*:*:*:*:*:*:*",
								VersionStartIncluding: "1.11.3",
								VersionEndIncluding:   "1.11.7",
							},
						},
					},
				},
			},
		},
	}

	var countAffectedClusters, countFixableCVEs int
	for _, cve := range cves {
		if isClusterAffectedByCVE(cluster, &cve) {
			countAffectedClusters++
		}
		if isK8sCVEFixable(&cve) {
			countFixableCVEs++
		}
	}
	assert.Equal(t, countAffectedClusters, 3)
	assert.Equal(t, countFixableCVEs, 1)
}
