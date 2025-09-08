package image_vulnerabilities

import (
	"context"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	deploymentMockDS "github.com/stackrox/rox/central/deployment/datastore/mocks"
	"github.com/stackrox/rox/central/metrics/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/grpc/authn/basic"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

type testImage storage.Image

func makeTestImage(_ *testing.T, id string) *testImage {
	return &testImage{Id: id}
}

func (i *testImage) withCVE(cve ...*storage.EmbeddedVulnerability) *testImage {
	if i.Scan == nil {
		i.Scan = &storage.ImageScan{
			OperatingSystem: "os",
			Components:      []*storage.EmbeddedImageScanComponent{{}}}
	}
	is := i.Scan
	is.Components[0].Vulns = append(is.Components[0].Vulns, cve...)
	return i
}

func (i *testImage) withTags(tags ...string) *testImage {
	for _, tag := range tags {
		i.Names = append(i.Names, &storage.ImageName{
			Remote:   "remote",
			Tag:      tag,
			Registry: "registry",
		})
	}
	return i
}

func getTestData(t *testing.T) ([]*storage.Deployment, map[string][]*storage.Image) {
	cves := getTestCVEs(t)

	images := getTestImages(t, cves)

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

func getTestImages(t *testing.T, cves []*storage.EmbeddedVulnerability) []*storage.Image {
	return []*storage.Image{
		(*storage.Image)(makeTestImage(t, "image-0").withTags("tag").withCVE(cves[0])),
		(*storage.Image)(makeTestImage(t, "image-1").withTags("tag").withCVE(cves[0], cves[1])),
		(*storage.Image)(makeTestImage(t, "image-2").withTags("tag").withCVE(cves[1])),
		(*storage.Image)(makeTestImage(t, "image-3").withTags("tag", "latest").withCVE(cves[1], cves[2])),
	}
}

func getTestCVEs(*testing.T) []*storage.EmbeddedVulnerability {
	return []*storage.EmbeddedVulnerability{
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
}

type labelsTotal struct {
	labels prometheus.Labels
	total  int
}

func TestQueryDeploymentsAndImages(t *testing.T) {
	ctrl := gomock.NewController(t)
	ds := deploymentMockDS.NewMockDataStore(ctrl)

	deployments, deploymentImages := getTestData(t)

	ds.EXPECT().WalkByQuery(gomock.Any(), gomock.Any(), gomock.Any()).
		Times(1).
		Do(func(_ context.Context, _ *v1.Query, f func(*storage.Deployment) error) {
			for _, deployment := range deployments {
				_ = f(deployment)
			}
		}).
		Return(nil)

	for _, deployment := range deployments {
		ds.EXPECT().GetImagesForDeployment(gomock.Any(), deployment).
			Times(1).Return(deploymentImages[deployment.Id], nil)
	}

	var actual = make(map[string][]*labelsTotal)

	mr := mocks.NewMockCustomRegistry(ctrl)
	tracker := New(mr, ds)
	mr.EXPECT().RegisterMetric(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Times(3)
	mr.EXPECT().SetTotal(gomock.Any(), gomock.Any(), gomock.Any()).
		AnyTimes().Do(
		func(metric string, labels prometheus.Labels, total int) {
			actual[metric] = append(actual[metric], &labelsTotal{labels, total})
		},
	)
	mr.EXPECT().Lock()
	mr.EXPECT().Reset("Severity_count")
	mr.EXPECT().Reset("Cluster_Namespace_Severity_count")
	mr.EXPECT().Reset("Deployment_ImageTag_count")
	mr.EXPECT().Unlock()

	cfg, err := tracker.NewConfiguration(
		&storage.PrometheusMetrics_Group{
			GatheringPeriodMinutes: 121,
			Descriptors: map[string]*storage.PrometheusMetrics_Group_Labels{
				"Severity_count": {
					Labels: []string{"Severity"},
				},
				"Cluster_Namespace_Severity_count": {
					Labels: []string{"Cluster", "Namespace", "Severity"},
				},
				"Deployment_ImageTag_count": {
					Labels: []string{"Deployment", "ImageTag"},
				}},
		})

	assert.NoError(t, err)
	tracker.Reconfigure(cfg)
	tracker.Gather(makeAdminContext(t))

	expected := map[string][]*labelsTotal{
		"Severity_count": {
			{prometheus.Labels{"Severity": "CRITICAL_VULNERABILITY_SEVERITY"}, 3},
			{prometheus.Labels{"Severity": "MODERATE_VULNERABILITY_SEVERITY"}, 4},
			{prometheus.Labels{"Severity": "LOW_VULNERABILITY_SEVERITY"}, 2},
		},
		"Cluster_Namespace_Severity_count": {
			{prometheus.Labels{"Cluster": "cluster-1", "Namespace": "namespace-1", "Severity": "CRITICAL_VULNERABILITY_SEVERITY"}, 1},
			{prometheus.Labels{"Cluster": "cluster-1", "Namespace": "namespace-2", "Severity": "CRITICAL_VULNERABILITY_SEVERITY"}, 2},
			{prometheus.Labels{"Cluster": "cluster-1", "Namespace": "namespace-2", "Severity": "MODERATE_VULNERABILITY_SEVERITY"}, 2},
			{prometheus.Labels{"Cluster": "cluster-2", "Namespace": "namespace-2", "Severity": "MODERATE_VULNERABILITY_SEVERITY"}, 2},
			{prometheus.Labels{"Cluster": "cluster-2", "Namespace": "namespace-2", "Severity": "LOW_VULNERABILITY_SEVERITY"}, 2},
		},
		"Deployment_ImageTag_count": {
			{prometheus.Labels{"Deployment": "D0", "ImageTag": "tag"}, 1},
			{prometheus.Labels{"Deployment": "D1", "ImageTag": "tag"}, 3},
			{prometheus.Labels{"Deployment": "D2", "ImageTag": "tag"}, 1},
			{prometheus.Labels{"Deployment": "D3", "ImageTag": "tag"}, 2},
			{prometheus.Labels{"Deployment": "D3", "ImageTag": "latest"}, 2},
		},
	}

	for metric := range expected {
		assert.Contains(t, actual, metric)
	}
	for metric, records := range actual {
		assert.ElementsMatch(t, expected[metric], records, metric)
	}
}

func Test_forEachImageVuln(t *testing.T) {
	i := 0
	interrupt := func(*finding) bool { i++; return false }
	pass := func(*finding) bool { i++; return true }

	t.Run("no panic on empty finding", func(t *testing.T) {
		i = 0
		assert.True(t, forEachImageVuln(interrupt, &finding{}))
		assert.Zero(t, i)
		assert.True(t, forEachImageVuln(pass, &finding{}))
		assert.Zero(t, i)
	})

	t.Run("interruption on false", func(t *testing.T) {
		i = 0

		image := makeTestImage(t, "test")
		image.withTags("test")
		cves := getTestCVEs(t)
		image.withCVE(cves...)

		assert.True(t, forEachImageVuln(pass, &finding{
			image: (*storage.Image)(image),
		}))
		assert.Equal(t, len(cves), i)

		assert.False(t, forEachImageVuln(interrupt, &finding{
			image: (*storage.Image)(image),
		}))
		assert.Equal(t, len(cves)+1, i)
	})
}

func makeAdminContext(t *testing.T) context.Context {
	authProvider, _ := authproviders.NewProvider(
		authproviders.WithEnabled(true),
		authproviders.WithID(uuid.NewDummy().String()),
		authproviders.WithName("Test Auth Provider"),
	)
	return basic.ContextWithAdminIdentity(t, authProvider)
}
