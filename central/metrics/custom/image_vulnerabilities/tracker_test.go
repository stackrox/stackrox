package image_vulnerabilities

import (
	"context"
	"io"
	"net/http/httptest"
	"testing"

	deploymentMockDS "github.com/stackrox/rox/central/deployment/datastore/mocks"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/metrics/custom/tracker"
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

	tracker := New(ds)

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

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(makeAdminContext(t),
		"GET", "/metrics", nil)

	r, err := metrics.GetCustomRegistry("Admin")
	if assert.NoError(t, err) {
		r.ServeHTTP(rec, req)
	}

	result := rec.Result()
	assert.Equal(t, 200, result.StatusCode)
	body, err := io.ReadAll(result.Body)
	_ = result.Body.Close()
	assert.NoError(t, err)
	assert.Equal(t,
		`# HELP rox_central_image_vuln_Cluster_Namespace_Severity_count The total number of image vulnerabilities aggregated by Cluster,Namespace,Severity and gathered every 2h1m0s
# TYPE rox_central_image_vuln_Cluster_Namespace_Severity_count gauge
rox_central_image_vuln_Cluster_Namespace_Severity_count{Cluster="cluster-1",Namespace="namespace-1",Severity="CRITICAL_VULNERABILITY_SEVERITY"} 1
rox_central_image_vuln_Cluster_Namespace_Severity_count{Cluster="cluster-1",Namespace="namespace-2",Severity="CRITICAL_VULNERABILITY_SEVERITY"} 2
rox_central_image_vuln_Cluster_Namespace_Severity_count{Cluster="cluster-1",Namespace="namespace-2",Severity="MODERATE_VULNERABILITY_SEVERITY"} 2
rox_central_image_vuln_Cluster_Namespace_Severity_count{Cluster="cluster-2",Namespace="namespace-2",Severity="LOW_VULNERABILITY_SEVERITY"} 2
rox_central_image_vuln_Cluster_Namespace_Severity_count{Cluster="cluster-2",Namespace="namespace-2",Severity="MODERATE_VULNERABILITY_SEVERITY"} 2
# HELP rox_central_image_vuln_Deployment_ImageTag_count The total number of image vulnerabilities aggregated by Deployment,ImageTag and gathered every 2h1m0s
# TYPE rox_central_image_vuln_Deployment_ImageTag_count gauge
rox_central_image_vuln_Deployment_ImageTag_count{Deployment="D0",ImageTag="tag"} 1
rox_central_image_vuln_Deployment_ImageTag_count{Deployment="D1",ImageTag="tag"} 3
rox_central_image_vuln_Deployment_ImageTag_count{Deployment="D2",ImageTag="tag"} 1
rox_central_image_vuln_Deployment_ImageTag_count{Deployment="D3",ImageTag="latest"} 2
rox_central_image_vuln_Deployment_ImageTag_count{Deployment="D3",ImageTag="tag"} 2
# HELP rox_central_image_vuln_Severity_count The total number of image vulnerabilities aggregated by Severity and gathered every 2h1m0s
# TYPE rox_central_image_vuln_Severity_count gauge
rox_central_image_vuln_Severity_count{Severity="CRITICAL_VULNERABILITY_SEVERITY"} 3
rox_central_image_vuln_Severity_count{Severity="LOW_VULNERABILITY_SEVERITY"} 2
rox_central_image_vuln_Severity_count{Severity="MODERATE_VULNERABILITY_SEVERITY"} 4
`,
		string(body))
}

func Test_forEachImageVuln(t *testing.T) {
	i := 0
	interrupt := func(*finding, error) bool { i++; return false }
	pass := func(*finding, error) bool { i++; return true }

	t.Run("no panic on empty finding", func(t *testing.T) {
		i = 0
		assert.NoError(t, forEachImageVuln(tracker.NewFindingCollector(interrupt), &finding{}))
		assert.Zero(t, i)
		assert.NoError(t, forEachImageVuln(tracker.NewFindingCollector(pass), &finding{}))
		assert.Zero(t, i)
	})

	t.Run("interruption on false", func(t *testing.T) {
		i = 0

		image := makeTestImage(t, "test")
		image.withTags("test")
		cves := getTestCVEs(t)
		image.withCVE(cves...)

		assert.NoError(t, forEachImageVuln(tracker.NewFindingCollector(pass), &finding{
			image: (*storage.Image)(image),
		}))
		assert.Equal(t, len(cves), i)

		assert.Error(t, forEachImageVuln(tracker.NewFindingCollector(interrupt), &finding{
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
