package telemetry

import (
	"context"
	"testing"

	"github.com/gobwas/glob"
	deploymentDS "github.com/stackrox/rox/central/deployment/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

type image storage.Image

func (i *image) withCVE(cve ...*storage.EmbeddedVulnerability) *image {
	if i.Scan == nil {
		i.Scan = &storage.ImageScan{
			OperatingSystem: "os",
			Components:      []*storage.EmbeddedImageScanComponent{{}}}
	}
	is := i.Scan
	is.Components[0].Vulns = append(is.Components[0].Vulns, cve...)
	return i
}

func makeTestImage(id string) *image {
	return &image{Id: id}
}

func (i *image) withTags(tags ...string) *image {
	for _, tag := range tags {
		i.Names = append(i.Names, &storage.ImageName{
			Remote:   "remote",
			Tag:      tag,
			Registry: "registry",
		})
	}
	return i
}

func Test_fetchMetrics(t *testing.T) {
	ctrl := gomock.NewController(t)
	ds := deploymentDS.NewMockDataStore(ctrl)

	globCache = make(map[string]glob.Glob)
	keysMap = parseAggregationKeys("Severity|Cluster,Namespace,Severity|Deployment=*4,ImageTag=latest")

	deployments := []*storage.Deployment{
		{Id: "deployment-1", Name: "D1", Namespace: "namespace-1", ClusterName: "cluster-1"},
		{Id: "deployment-2", Name: "D2", Namespace: "namespace-2", ClusterName: "cluster-1"},
		{Id: "deployment-3", Name: "D3", Namespace: "namespace-2", ClusterName: "cluster-1"},
		{Id: "deployment-4", Name: "D4", Namespace: "namespace-2", ClusterName: "cluster-2"},
	}

	cves := []*storage.EmbeddedVulnerability{
		{Cve: "cve-0", Cvss: 7.5,
			CvssV3:   &storage.CVSSV3{Severity: storage.CVSSV3_CRITICAL},
			Severity: storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
		},
		{Cve: "cve-1", Cvss: 5.0,
			CvssV3:   &storage.CVSSV3{Severity: storage.CVSSV3_MEDIUM},
			Severity: storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
		},
		{Cve: "cve-2", Cvss: 3.0,
			CvssV3:   &storage.CVSSV3{Severity: storage.CVSSV3_LOW},
			Severity: storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY,
		},
	}

	images := []*storage.Image{
		(*storage.Image)(makeTestImage("image-0").withTags("tag").withCVE(cves[0])),
		(*storage.Image)(makeTestImage("image-1").withTags("tag").withCVE(cves[0], cves[1])),
		(*storage.Image)(makeTestImage("image-2").withTags("tag").withCVE(cves[1])),
		(*storage.Image)(makeTestImage("image-3").withTags("tag", "latest").withCVE(cves[1], cves[2])),
	}

	deploymentImages := map[string][]*storage.Image{
		"deployment-1": {images[0]},
		"deployment-2": {images[0], images[1]},
		"deployment-3": {images[2]},
		"deployment-4": {images[3]},
	}

	ds.EXPECT().WalkByQuery(gomock.Any(), gomock.Any(), gomock.Any()).
		Times(1).
		Do(func(ctx context.Context, q *v1.Query, f func(deployment *storage.Deployment) error) {
			for _, deployment := range deployments {
				_ = f(deployment)
			}
		}).
		Return(nil)

	for _, deployment := range deployments {
		ds.EXPECT().GetImagesForDeployment(gomock.Any(), deployment).
			Times(1).Return(deploymentImages[deployment.Id], nil)
	}
	type cve = string
	type labels = map[string]string

	var aggregated map[aggregationKey]map[keyInstance]int
	i := &trackImpl{ds: ds,
		aggregated: func(a map[aggregationKey]map[keyInstance]int) {
			aggregated = a
		}}

	i.trackCvssMetrics(context.Background())
	assert.Equal(t, map[aggregationKey]map[keyInstance]int{
		"Severity": map[string]int{
			"CRITICAL_VULNERABILITY_SEVERITY": 3,
			"LOW_VULNERABILITY_SEVERITY":      2,
			"MODERATE_VULNERABILITY_SEVERITY": 4,
		},
		"Cluster,Namespace,Severity": map[string]int{
			"cluster-1|namespace-1|CRITICAL_VULNERABILITY_SEVERITY": 1,
			"cluster-1|namespace-2|CRITICAL_VULNERABILITY_SEVERITY": 2,
			"cluster-1|namespace-2|MODERATE_VULNERABILITY_SEVERITY": 2,
			"cluster-2|namespace-2|LOW_VULNERABILITY_SEVERITY":      2,
			"cluster-2|namespace-2|MODERATE_VULNERABILITY_SEVERITY": 2,
		},
		"Deployment=*4,ImageTag=latest": map[string]int{
			"D4|latest": 2,
		},
	}, aggregated)
}
