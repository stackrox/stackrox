package telemetry

import (
	"context"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
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
	ds := deploymentDS.NewMockDataStore(ctrl)

	deployments, deploymentImages := getTestData()

	ds.EXPECT().WalkByQuery(gomock.Any(), gomock.Any(), gomock.Any()).
		Times(1).
		Do(func(_ context.Context, _ *v1.Query, f func(deployment *storage.Deployment) error) {
			for _, deployment := range deployments {
				_ = f(deployment)
			}
		}).
		Return(nil)

	for _, deployment := range deployments {
		ds.EXPECT().GetImagesForDeployment(gomock.Any(), deployment).
			Times(1).Return(deploymentImages[deployment.Id], nil)
	}

	var actual = make(map[metricName][]*record)
	metricExpressions := map[metricName]map[Label][]*expression{
		"Severity_total": {
			"Severity": nil,
		},
		"Cluster_Namespace_Severity_total": {
			"Cluster":   nil,
			"Namespace": nil,
			"Severity":  {},
		},
		"Deployment_ImageTag_total": {
			"Deployment": {{"=", "*3"}},
			"ImageTag":   {{"=", "latest"}},
		},
	}

	a := aggregator{
		ds: ds,
		trackFunc: func(metric string, labels prometheus.Labels, total int) {
			actual[metricName(metric)] = append(actual[metricName(metric)],
				&record{
					labels: labels,
					total:  total,
				})
		},
	}

	a.track(context.Background(), metricExpressions)

	expected := map[metricName][]*record{
		"Severity_total": {
			{prometheus.Labels{"Severity": "CRITICAL_VULNERABILITY_SEVERITY"}, 3},
			{prometheus.Labels{"Severity": "MODERATE_VULNERABILITY_SEVERITY"}, 4},
			{prometheus.Labels{"Severity": "LOW_VULNERABILITY_SEVERITY"}, 2},
		},
		"Cluster_Namespace_Severity_total": {
			{prometheus.Labels{"Cluster": "cluster-1", "Namespace": "namespace-1", "Severity": "CRITICAL_VULNERABILITY_SEVERITY"}, 1},
			{prometheus.Labels{"Cluster": "cluster-1", "Namespace": "namespace-2", "Severity": "CRITICAL_VULNERABILITY_SEVERITY"}, 2},
			{prometheus.Labels{"Cluster": "cluster-1", "Namespace": "namespace-2", "Severity": "MODERATE_VULNERABILITY_SEVERITY"}, 2},
			{prometheus.Labels{"Cluster": "cluster-2", "Namespace": "namespace-2", "Severity": "MODERATE_VULNERABILITY_SEVERITY"}, 2},
			{prometheus.Labels{"Cluster": "cluster-2", "Namespace": "namespace-2", "Severity": "LOW_VULNERABILITY_SEVERITY"}, 2},
		},
		"Deployment_ImageTag_total": {
			{prometheus.Labels{"Deployment": "D3", "ImageTag": "latest"}, 2},
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
