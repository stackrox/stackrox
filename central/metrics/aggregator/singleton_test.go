package aggregator

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/pkg/errors"
	configDS "github.com/stackrox/rox/central/config/datastore/mocks"
	deploymentDS "github.com/stackrox/rox/central/deployment/datastore/mocks"
	"github.com/stackrox/rox/central/metrics"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func Test_getRegistryName(t *testing.T) {
	metrics.GetExternalRegistry("r1")
	metrics.GetExternalRegistry("r2")

	runner := &aggregatorRunner{}

	for path, expected := range map[string]string{
		"/metrics/r1":     "r1",
		"/metrics/r2":     "r2",
		"/metrics/r1?a=b": "r1",
		"/metrics/r2?a=b": "r2",
		"/metrics":        "",
	} {
		u, _ := url.Parse("https://central" + path)
		name, ok := runner.getRegistryName(&http.Request{URL: u})
		assert.True(t, ok)
		assert.Equal(t, expected, name)
	}

	for _, path := range []string{
		"",
		"/r1",
		"/r1/",
		"/metrics/r1/",
		"/metricsr1",
		"/metricsr1/",
		"/metricsr1/r1",
		"/metrics/bad",
		"/kilometrics/r1",
	} {
		u, _ := url.Parse("https://central" + path)
		name, ok := runner.getRegistryName(&http.Request{URL: u})
		assert.False(t, ok)
		assert.Empty(t, name)
	}
}

func TestRunner_makeRunner(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Run("nil configuration", func(t *testing.T) {
		cds := configDS.NewMockDataStore(ctrl)
		cds.EXPECT().GetPrivateConfig(gomock.Any()).Times(1).Return(
			&storage.PrivateConfig{
				PrometheusMetricsConfig: nil,
			},
			nil)
		runner := makeRunner(nil)
		runner.initialize(cds)
		assert.NotNil(t, runner)

		// The synchronous loop should exit on nil ticker.
		runner.image_vulnerabilities.Run(runner.ctx)

		cfg, err := runner.ParseConfiguration(nil)
		assert.NoError(t, err)
		assert.NotNil(t, cfg)
		runner.Reconfigure(&RunnerConfiguration{})

		// The synchronous loop should exit on nil ticker.
		runner.image_vulnerabilities.Run(runner.ctx)
	})

	t.Run("err configuration", func(t *testing.T) {
		cds := configDS.NewMockDataStore(ctrl)
		cds.EXPECT().GetPrivateConfig(gomock.Any()).Times(1).Return(
			nil,
			errors.New("DB error"))
		runner := makeRunner(nil)
		assert.NotNil(t, runner)
		runner.initialize(cds)

		// The synchronous loop should exit on nil ticker.
		runner.image_vulnerabilities.Run(runner.ctx)

		cfg, err := runner.ParseConfiguration(nil)
		assert.NoError(t, err)
		assert.NotNil(t, cfg)
		runner.Reconfigure(&RunnerConfiguration{})

		// The synchronous loop should exit on nil ticker.
		runner.image_vulnerabilities.Run(runner.ctx)
	})
}

func TestRunner_ServeHTTP(t *testing.T) {
	ctrl := gomock.NewController(t)

	cds := configDS.NewMockDataStore(ctrl)
	cds.EXPECT().GetPrivateConfig(gomock.Any()).Times(1).Return(
		&storage.PrivateConfig{
			PrometheusMetricsConfig: &storage.PrometheusMetricsConfig{
				ImageVulnerabilities: &storage.PrometheusMetricsConfig_Metrics{
					GatheringPeriodMinutes: 10,
					Metrics: map[string]*storage.PrometheusMetricsConfig_Labels{
						"metric1": {
							Labels: map[string]*storage.PrometheusMetricsConfig_Labels_Expression{
								"Cluster":  nil,
								"Severity": nil,
							},
							Exposure:     storage.PrometheusMetricsConfig_Labels_EXTERNAL,
							RegistryName: "custom",
						}}}}},
		nil)

	dds := deploymentDS.NewMockDataStore(ctrl)

	dds.EXPECT().WalkByQuery(gomock.Any(), gomock.Any(), gomock.Any()).
		Times(1).
		Do(func(_ context.Context, _ *v1.Query, f func(*storage.Deployment) error) {
			_ = f(&storage.Deployment{
				Name:        "deployment1",
				ClusterName: "cluster1",
			})
		}).
		Return(nil)

	dds.EXPECT().GetImagesForDeployment(gomock.Any(), gomock.Any()).
		Times(1).Return([]*storage.Image{{
		Names: []*storage.ImageName{{FullName: "fullname1"}},
		Scan: &storage.ImageScan{
			Components: []*storage.EmbeddedImageScanComponent{{
				Vulns: []*storage.EmbeddedVulnerability{{
					Severity: storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY,
				}},
			}},
		}},
	}, nil)

	runner := makeRunner(dds)
	runner.initialize(cds)
	runner.Stop()
	runner.image_vulnerabilities.Run(runner.ctx)
	rec := httptest.NewRecorder()
	runner.ServeHTTP(rec, httptest.NewRequest("GET", "/metrics/custom", nil))

	result := rec.Result()
	assert.Equal(t, 200, result.StatusCode)
	body, err := io.ReadAll(result.Body)
	_ = result.Body.Close()
	assert.NoError(t, err)
	assert.Equal(t,
		`# HELP rox_central_metric1 The total number of aggregated CVEs aggregated by Cluster,Severity and gathered every 10m0s`+"\n"+
			`# TYPE rox_central_metric1 gauge`+"\n"+
			`rox_central_metric1{Cluster="cluster1",Severity="IMPORTANT_VULNERABILITY_SEVERITY"} 1`+"\n",
		string(body))
}
