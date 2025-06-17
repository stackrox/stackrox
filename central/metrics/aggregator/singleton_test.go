package aggregator

import (
	"context"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/pkg/errors"
	configDS "github.com/stackrox/rox/central/config/datastore/mocks"
	deploymentDS "github.com/stackrox/rox/central/deployment/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestRunner_makeRunner(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Run("nil configuration", func(t *testing.T) {
		cds := configDS.NewMockDataStore(ctrl)
		cds.EXPECT().GetPrivateConfig(gomock.Any()).Times(1).Return(
			&storage.PrivateConfig{
				Metrics: nil,
			},
			nil)
		runner := makeRunner(nil)
		runner.initialize(cds)
		assert.NotNil(t, runner)

		// The synchronous loop should exit on nil ticker.
		runner.image_vulnerabilities.Run(runner.ctx)

		cfg, err := runner.ValidateConfiguration(nil)
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

		cfg, err := runner.ValidateConfiguration(nil)
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
			Metrics: &storage.PrometheusMetrics{
				ImageVulnerabilities: &storage.PrometheusMetrics_MetricGroup{
					GatheringPeriodMinutes: 10,
					Metrics: map[string]*storage.PrometheusMetrics_MetricGroup_Labels{
						"test_metric": {
							Labels: []string{"Cluster", "Severity"},
						},
					}}}},
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

	expectedBody := func(metricName string) string {
		return `# HELP rox_central_` + metricName + ` The total number of aggregated CVEs aggregated by Cluster,Severity and gathered every 10m0s` + "\n" +
			`# TYPE rox_central_` + metricName + ` gauge` + "\n" +
			`rox_central_` + metricName + `{Cluster="cluster1",Severity="IMPORTANT_VULNERABILITY_SEVERITY"} 1` + "\n"
	}

	t.Run("body", func(t *testing.T) {
		rec := httptest.NewRecorder()
		runner.ServeHTTP(rec, httptest.NewRequest("GET", "/metrics/custom", nil))

		result := rec.Result()
		assert.Equal(t, 200, result.StatusCode)
		body, err := io.ReadAll(result.Body)
		_ = result.Body.Close()
		assert.NoError(t, err)
		assert.Equal(t, expectedBody("test_metric"), string(body))
	})
}
