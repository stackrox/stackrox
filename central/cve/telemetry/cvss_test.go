package telemetry

import (
	"context"
	"testing"

	deploymentDS "github.com/stackrox/rox/central/deployment/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

type image storage.Image

func (i *image) withCVE(cve string, severityV3 storage.CVSSV3_Severity, severity storage.VulnerabilitySeverity) *image {
	if i.Scan == nil {
		i.Scan = &storage.ImageScan{
			OperatingSystem: "os",
			Components:      []*storage.EmbeddedImageScanComponent{{}}}
	}
	is := i.Scan
	is.Components[0].Vulns = append(is.Components[0].Vulns, &storage.EmbeddedVulnerability{
		Cve:  cve,
		Cvss: 2.5,
		CvssV3: &storage.CVSSV3{
			Severity: severityV3,
		},
		Severity: severity,
	})
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

	deployments := []*storage.Deployment{
		{Id: "deployment-1", Namespace: "namespace-1", ClusterName: "cluster-1"},
		{Id: "deployment-2", Namespace: "namespace-2", ClusterName: "cluster-1"},
		{Id: "deployment-3", Namespace: "namespace-2", ClusterName: "cluster-1"},
		{Id: "deployment-4", Namespace: "namespace-2", ClusterName: "cluster-2"},
	}

	images := map[string][]*storage.Image{
		"deployment-1": {
			(*storage.Image)(makeTestImage("image-1").withTags("tag").withCVE(
				"cve-1", storage.CVSSV3_CRITICAL,
				storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY)),
		},
		"deployment-2": {
			(*storage.Image)(makeTestImage("image-2").withTags("tag").withCVE(
				"cve-1", storage.CVSSV3_CRITICAL,
				storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY)),
		},
		"deployment-3": {
			(*storage.Image)(makeTestImage("image-2").withTags("tag").withCVE(
				"cve-1", storage.CVSSV3_CRITICAL,
				storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY)),
		},
		"deployment-4": {
			(*storage.Image)(makeTestImage("image-3").withTags("tag1", "tag2").withCVE(
				"cve-2", storage.CVSSV3_HIGH,
				storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY)),
		},
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
			Times(1).Return(images[deployment.Id], nil)
	}
	type cve = string
	type labels = map[string]string

	results := []labels{}
	var severitySum float64
	var aggregated map[string]int
	i := &trackImpl{ds: ds,
		aggregated: func(a map[string]int) {
			aggregated = a
		},
		cvssGauge: func(l labels, f float64) {
			results = append(results, l)
			severitySum += f
		}}

	i.trackCvssMetrics(context.Background())
	assert.Equal(t, map[string]int{
		"LOW_VULNERABILITY_SEVERITY":       1,
		"MODERATE_VULNERABILITY_SEVERITY":  1,
		"IMPORTANT_VULNERABILITY_SEVERITY": 1,
		"CRITICAL_VULNERABILITY_SEVERITY":  2,
	}, aggregated)

	assert.Equal(t, 12.5, severitySum)
	assert.Equal(t, []labels{
		{
			"CVE":             "cve-1",
			"Cluster":         "cluster-1",
			"ImageId":         "image-1",
			"ImageRegistry":   "registry",
			"ImageRemote":     "remote",
			"ImageTag":        "tag",
			"IsFixable":       "false",
			"Namespace":       "namespace-1",
			"OperatingSystem": "os",
			"SeverityV2":      "UNKNOWN",
			"SeverityV3":      "CRITICAL",
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
			"SeverityV2":      "UNKNOWN",
			"SeverityV3":      "CRITICAL",
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
			"SeverityV2":      "UNKNOWN",
			"SeverityV3":      "CRITICAL",
		}, {
			"CVE":             "cve-2",
			"Cluster":         "cluster-2",
			"ImageId":         "image-3",
			"ImageRegistry":   "registry",
			"ImageRemote":     "remote",
			"ImageTag":        "tag1",
			"IsFixable":       "false",
			"Namespace":       "namespace-2",
			"OperatingSystem": "os",
			"SeverityV2":      "UNKNOWN",
			"SeverityV3":      "HIGH",
		}, {"CVE": "cve-2",
			"Cluster":         "cluster-2",
			"ImageId":         "image-3",
			"ImageRegistry":   "registry",
			"ImageRemote":     "remote",
			"ImageTag":        "tag2",
			"IsFixable":       "false",
			"Namespace":       "namespace-2",
			"OperatingSystem": "os",
			"SeverityV2":      "UNKNOWN",
			"SeverityV3":      "HIGH",
		},
	}, results)
}
