package telemetry

import (
	"context"
	"testing"

	deploymentDS "github.com/stackrox/rox/central/deployment/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
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

	t.Setenv(env.EnableCVSSMetrics.EnvVar(), "true")
	keysMap = parseAggregationKeys("Severity|Cluster,Namespace,Severity")

	deployments := []*storage.Deployment{
		{Id: "deployment-1", Namespace: "namespace-1", ClusterName: "cluster-1"},
		{Id: "deployment-2", Namespace: "namespace-2", ClusterName: "cluster-1"},
		{Id: "deployment-3", Namespace: "namespace-2", ClusterName: "cluster-1"},
		{Id: "deployment-4", Namespace: "namespace-2", ClusterName: "cluster-2"},
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

	results := []labels{}
	var severitySum float64
	var aggregated map[aggregationKey]map[keyInstance]int
	i := &trackImpl{ds: ds,
		aggregated: func(a map[aggregationKey]map[keyInstance]int) {
			aggregated = a
		},
		cvssGauge: func(l labels, f float64) {
			results = append(results, l)
			severitySum += f
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
	}, aggregated)

	assert.Equal(t, 48.5, severitySum)
	assert.Equal(t, []labels{
		{
			"CVE":             "cve-0",
			"Cluster":         "cluster-1",
			"ImageId":         "image-0",
			"ImageRegistry":   "registry",
			"ImageRemote":     "remote",
			"ImageTag":        "tag",
			"IsFixable":       "false",
			"Namespace":       "namespace-1",
			"OperatingSystem": "os",
			"Severity":        "CRITICAL_VULNERABILITY_SEVERITY",
			"SeverityV2":      "UNKNOWN",
			"SeverityV3":      "CRITICAL",
		}, {
			"CVE":             "cve-0",
			"Cluster":         "cluster-1",
			"ImageId":         "image-0",
			"ImageRegistry":   "registry",
			"ImageRemote":     "remote",
			"ImageTag":        "tag",
			"IsFixable":       "false",
			"Namespace":       "namespace-2",
			"OperatingSystem": "os",
			"Severity":        "CRITICAL_VULNERABILITY_SEVERITY",
			"SeverityV2":      "UNKNOWN",
			"SeverityV3":      "CRITICAL",
		}, {
			"CVE":             "cve-0",
			"Cluster":         "cluster-1",
			"ImageId":         "image-1",
			"ImageRegistry":   "registry",
			"ImageRemote":     "remote",
			"ImageTag":        "tag",
			"IsFixable":       "false",
			"Namespace":       "namespace-2",
			"OperatingSystem": "os",
			"Severity":        "CRITICAL_VULNERABILITY_SEVERITY",
			"SeverityV2":      "UNKNOWN",
			"SeverityV3":      "CRITICAL",
		}, {
			"CVE":             "cve-1",
			"Cluster":         "cluster-1",
			"ImageId":         "image-1",
			"ImageRegistry":   "registry",
			"ImageRemote":     "remote",
			"ImageTag":        "tag",
			"IsFixable":       "false",
			"Namespace":       "namespace-2",
			"OperatingSystem": "os",
			"Severity":        "MODERATE_VULNERABILITY_SEVERITY",
			"SeverityV2":      "UNKNOWN",
			"SeverityV3":      "MEDIUM",
		}, {
			"CVE":             "cve-1",
			"Cluster":         "cluster-1",
			"ImageId":         "image-2",
			"ImageRegistry":   "registry",
			"ImageRemote":     "remote",
			"ImageTag":        "tag",
			"IsFixable":       "false",
			"Namespace":       "namespace-2",
			"OperatingSystem": "os",
			"Severity":        "MODERATE_VULNERABILITY_SEVERITY",
			"SeverityV2":      "UNKNOWN",
			"SeverityV3":      "MEDIUM",
		}, {
			"CVE":             "cve-1",
			"Cluster":         "cluster-2",
			"ImageId":         "image-3",
			"ImageRegistry":   "registry",
			"ImageRemote":     "remote",
			"ImageTag":        "tag",
			"IsFixable":       "false",
			"Namespace":       "namespace-2",
			"OperatingSystem": "os",
			"Severity":        "MODERATE_VULNERABILITY_SEVERITY",
			"SeverityV2":      "UNKNOWN",
			"SeverityV3":      "MEDIUM",
		},
		{
			"CVE":             "cve-1",
			"Cluster":         "cluster-2",
			"ImageId":         "image-3",
			"ImageRegistry":   "registry",
			"ImageRemote":     "remote",
			"ImageTag":        "latest",
			"IsFixable":       "false",
			"Namespace":       "namespace-2",
			"OperatingSystem": "os",
			"Severity":        "MODERATE_VULNERABILITY_SEVERITY",
			"SeverityV2":      "UNKNOWN",
			"SeverityV3":      "MEDIUM",
		}, {
			"CVE":             "cve-2",
			"Cluster":         "cluster-2",
			"ImageId":         "image-3",
			"ImageRegistry":   "registry",
			"ImageRemote":     "remote",
			"ImageTag":        "tag",
			"IsFixable":       "false",
			"Namespace":       "namespace-2",
			"OperatingSystem": "os",
			"Severity":        "LOW_VULNERABILITY_SEVERITY",
			"SeverityV2":      "UNKNOWN",
			"SeverityV3":      "LOW",
		}, {
			"CVE":             "cve-2",
			"Cluster":         "cluster-2",
			"ImageId":         "image-3",
			"ImageRegistry":   "registry",
			"ImageRemote":     "remote",
			"ImageTag":        "latest",
			"IsFixable":       "false",
			"Namespace":       "namespace-2",
			"OperatingSystem": "os",
			"Severity":        "LOW_VULNERABILITY_SEVERITY",
			"SeverityV2":      "UNKNOWN",
			"SeverityV3":      "LOW",
		},
	}, results)
}

func Test_keysMap(t *testing.T) {
	keys := parseAggregationKeys("Namespace,Severity,IsFixable|Cluster|SeverityV3")
	assert.Equal(t, map[aggregationKey][]string{
		"Cluster":                      {"Cluster"},
		"Namespace,Severity,IsFixable": {"Namespace", "Severity", "IsFixable"},
		"SeverityV3":                   {"SeverityV3"},
	}, keys)
}
