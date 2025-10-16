package image_vulnerabilities

import (
	"context"
	"io"
	"net/http/httptest"
	"testing"

	deploymentMockDS "github.com/stackrox/rox/central/deployment/datastore/mocks"
	"github.com/stackrox/rox/central/metrics"
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
		imageScan := &storage.ImageScan{}
		imageScan.SetOperatingSystem("os")
		imageScan.SetComponents([]*storage.EmbeddedImageScanComponent{{}})
		i.Scan = imageScan
	}
	is := i.Scan
	is.GetComponents()[0].SetVulns(append(is.GetComponents()[0].GetVulns(), cve...))
	return i
}

func (i *testImage) withTags(tags ...string) *testImage {
	for _, tag := range tags {
		imageName := &storage.ImageName{}
		imageName.SetRemote("remote")
		imageName.SetTag(tag)
		imageName.SetRegistry("registry")
		i.Names = append(i.Names, imageName)
	}
	return i
}

func getTestData(t *testing.T) ([]*storage.Deployment, map[string][]*storage.Image) {
	cves := getTestCVEs(t)

	images := getTestImages(t, cves)

	deployments := []*storage.Deployment{
		storage.Deployment_builder{Id: "deployment-0", Name: "D0", Namespace: "namespace-1", ClusterName: "cluster-1"}.Build(),
		storage.Deployment_builder{Id: "deployment-1", Name: "D1", Namespace: "namespace-2", ClusterName: "cluster-1"}.Build(),
		storage.Deployment_builder{Id: "deployment-2", Name: "D2", Namespace: "namespace-2", ClusterName: "cluster-1"}.Build(),
		storage.Deployment_builder{Id: "deployment-3", Name: "D3", Namespace: "namespace-2", ClusterName: "cluster-2"}.Build(),
	}

	deploymentImages := map[string][]*storage.Image{
		deployments[0].GetId(): {images[0]},
		deployments[1].GetId(): {images[0], images[1]},
		deployments[2].GetId(): {images[2]},
		deployments[3].GetId(): {images[3]},
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
		storage.EmbeddedVulnerability_builder{Cve: "cve-0", Cvss: 7.5,
			CvssV3:   storage.CVSSV3_builder{Severity: storage.CVSSV3_CRITICAL}.Build(),
			Severity: storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
		}.Build(),
		storage.EmbeddedVulnerability_builder{Cve: "cve-1", Cvss: 5.0,
			CvssV3:   storage.CVSSV3_builder{Severity: storage.CVSSV3_MEDIUM}.Build(),
			Severity: storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
		}.Build(),
		storage.EmbeddedVulnerability_builder{Cve: "cve-2", Cvss: 3.0,
			CvssV3:   storage.CVSSV3_builder{Severity: storage.CVSSV3_LOW}.Build(),
			Severity: storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY,
		}.Build(),
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
			Times(1).Return(deploymentImages[deployment.GetId()], nil)
	}

	tracker := New(ds)

	cfg, err := tracker.NewConfiguration(
		storage.PrometheusMetrics_Group_builder{
			GatheringPeriodMinutes: 121,
			Descriptors: map[string]*storage.PrometheusMetrics_Group_Labels{
				"Severity_count": storage.PrometheusMetrics_Group_Labels_builder{
					Labels: []string{"Severity"},
				}.Build(),
				"Cluster_Namespace_Severity_count": storage.PrometheusMetrics_Group_Labels_builder{
					Labels: []string{"Cluster", "Namespace", "Severity"},
				}.Build(),
				"Deployment_ImageTag_count": storage.PrometheusMetrics_Group_Labels_builder{
					Labels: []string{"Deployment", "ImageTag"},
				}.Build()},
		}.Build())

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
