package custom

import (
	"context"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/pkg/errors"
	configDS "github.com/stackrox/rox/central/config/datastore/mocks"
	deploymentDS "github.com/stackrox/rox/central/deployment/datastore/mocks"
	"github.com/stackrox/rox/central/metrics"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/grpc/authn/basic"
	"github.com/stackrox/rox/pkg/uuid"
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
		runner := makeRunner(metrics.MakeCustomRegistry(), nil)
		runner.initialize(cds)
		assert.NotNil(t, runner)

		ctx := context.Background()
		assert.NotPanics(t, func() {
			runner.image_vulnerabilities.Gather(ctx)
		})

		cfg, err := runner.ValidateConfiguration(nil)
		assert.NoError(t, err)
		assert.NotNil(t, cfg)
		runner.Reconfigure(&RunnerConfiguration{})

		assert.NotPanics(t, func() {
			runner.image_vulnerabilities.Gather(ctx)
		})
	})

	t.Run("err configuration", func(t *testing.T) {
		cds := configDS.NewMockDataStore(ctrl)
		cds.EXPECT().GetPrivateConfig(gomock.Any()).Times(1).Return(
			nil,
			errors.New("DB error"))
		runner := makeRunner(metrics.MakeCustomRegistry(), nil)
		assert.NotNil(t, runner)
		runner.initialize(cds)

		ctx := context.Background()
		assert.NotPanics(t, func() {
			runner.image_vulnerabilities.Gather(ctx)
		})

		cfg, err := runner.ValidateConfiguration(nil)
		assert.NoError(t, err)
		assert.NotNil(t, cfg)
		runner.Reconfigure(&RunnerConfiguration{})

		assert.NotPanics(t, func() {
			runner.image_vulnerabilities.Gather(ctx)
		})
	})
}

func TestRunner_ServeHTTP(t *testing.T) {
	ctrl := gomock.NewController(t)

	cds := configDS.NewMockDataStore(ctrl)

	cds.EXPECT().GetPrivateConfig(gomock.Any()).Times(1).Return(
		&storage.PrivateConfig{
			Metrics: &storage.PrometheusMetrics{
				ImageVulnerabilities: &storage.PrometheusMetrics_Group{
					GatheringPeriodMinutes: 10,
					Descriptors: map[string]*storage.PrometheusMetrics_Group_Labels{
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

	runner := makeRunner(metrics.MakeCustomRegistry(), dds)
	runner.initialize(cds)
	runner.image_vulnerabilities.Gather(makeAdminContext(t))

	expectedBody := func(metricName string) string {
		return `# HELP rox_central_` + metricName + ` The total number of aggregated CVEs aggregated by Cluster,Severity and gathered every 10m0s` + "\n" +
			`# TYPE rox_central_` + metricName + ` gauge` + "\n" +
			`rox_central_` + metricName + `{Cluster="cluster1",Severity="IMPORTANT_VULNERABILITY_SEVERITY"} 1` + "\n"
	}

	t.Run("body", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequestWithContext(makeAdminContext(t),
			"GET", "/metrics", nil)
		runner.ServeHTTP(rec, req)

		result := rec.Result()
		assert.Equal(t, 200, result.StatusCode)
		body, err := io.ReadAll(result.Body)
		_ = result.Body.Close()
		assert.NoError(t, err)
		assert.Equal(t, expectedBody("test_metric"), string(body))
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
