package vulnerabilities

import (
	"context"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	deploymentDS "github.com/stackrox/rox/central/deployment/datastore"
	deploymentMockDS "github.com/stackrox/rox/central/deployment/datastore/mocks"
	"github.com/stackrox/rox/central/metrics/aggregator/common"
	v1api "github.com/stackrox/rox/generated/api/v1"
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

func getTestData() ([]*storage.Deployment, map[string][]*storage.Image) {
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

	deployments := []*storage.Deployment{
		{Id: "deployment-0", Name: "D0", Namespace: "namespace-1", ClusterName: "cluster-1"},
		{Id: "deployment-1", Name: "D1", Namespace: "namespace-2", ClusterName: "cluster-1"},
		{Id: "deployment-2", Name: "D2", Namespace: "namespace-2", ClusterName: "cluster-1"},
		{Id: "deployment-3", Name: "D3", Namespace: "namespace-2", ClusterName: "cluster-2"},
	}

	deploymentImages := map[string][]*storage.Image{
		deployments[0].Id: {images[0]},
		deployments[1].Id: {images[0], images[1]},
		deployments[2].Id: {images[2]},
		deployments[3].Id: {images[3]},
	}
	return deployments, deploymentImages
}

func Test_track(t *testing.T) {
	ctrl := gomock.NewController(t)
	ds := deploymentMockDS.NewMockDataStore(ctrl)

	deployments, deploymentImages := getTestData()

	ds.EXPECT().WalkByQuery(gomock.Any(), gomock.Any(), gomock.Any()).
		Times(1).
		Do(func(_ context.Context, _ *v1api.Query, f func(deployment *storage.Deployment) error) {
			for _, deployment := range deployments {
				_ = f(deployment)
			}
		}).
		Return(nil)

	for _, deployment := range deployments {
		ds.EXPECT().GetImagesForDeployment(gomock.Any(), deployment).
			Times(1).Return(deploymentImages[deployment.Id], nil)
	}

	var actual = make(map[common.MetricName][]*common.Record)
	metricExpressions := map[common.MetricName]map[common.Label][]*common.Expression{
		"Severity_total": {
			"Severity": nil,
		},
		"Cluster_Namespace_Severity_total": {
			"Cluster":   nil,
			"Namespace": nil,
			"Severity":  {},
		},
		"Deployment_ImageTag_total": {
			"Deployment": {common.MustMakeExpression("=", "*3")},
			"ImageTag":   {common.MustMakeExpression("=", "latest")},
		},
	}

	a := common.MakeTrackWrapper[deploymentDS.DataStore](
		ds,
		func() common.MetricsConfig {
			return metricExpressions
		},
		TrackVulnerabilityMetrics)
	a.TrackFunc = func(metric string, labels prometheus.Labels, total int) {
		actual[common.MetricName(metric)] = append(actual[common.MetricName(metric)], common.MakeRecord(labels, total))
	}

	a.Track(context.Background())

	expected := map[common.MetricName][]*common.Record{
		"Severity_total": {
			common.MakeRecord(prometheus.Labels{"Severity": "CRITICAL_VULNERABILITY_SEVERITY"}, 3),
			common.MakeRecord(prometheus.Labels{"Severity": "MODERATE_VULNERABILITY_SEVERITY"}, 4),
			common.MakeRecord(prometheus.Labels{"Severity": "LOW_VULNERABILITY_SEVERITY"}, 2),
		},
		"Cluster_Namespace_Severity_total": {
			common.MakeRecord(prometheus.Labels{"Cluster": "cluster-1", "Namespace": "namespace-1", "Severity": "CRITICAL_VULNERABILITY_SEVERITY"}, 1),
			common.MakeRecord(prometheus.Labels{"Cluster": "cluster-1", "Namespace": "namespace-2", "Severity": "CRITICAL_VULNERABILITY_SEVERITY"}, 2),
			common.MakeRecord(prometheus.Labels{"Cluster": "cluster-1", "Namespace": "namespace-2", "Severity": "MODERATE_VULNERABILITY_SEVERITY"}, 2),
			common.MakeRecord(prometheus.Labels{"Cluster": "cluster-2", "Namespace": "namespace-2", "Severity": "MODERATE_VULNERABILITY_SEVERITY"}, 2),
			common.MakeRecord(prometheus.Labels{"Cluster": "cluster-2", "Namespace": "namespace-2", "Severity": "LOW_VULNERABILITY_SEVERITY"}, 2),
		},
		"Deployment_ImageTag_total": {
			common.MakeRecord(prometheus.Labels{"Deployment": "D3", "ImageTag": "latest"}, 2),
		},
	}

	for metric := range expected {
		assert.Contains(t, actual, metric)
	}
	for metric, records := range actual {
		if assert.Len(t, expected[metric], len(records), metric) {
			for i, record := range records {
				assert.Contains(t, expected[metric], record, "metric: %s, record: %d", metric, i)
			}
		}
	}
}
