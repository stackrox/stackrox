package custom

import (
	"context"
	"fmt"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/pkg/errors"
	alertDS "github.com/stackrox/rox/central/alert/datastore/mocks"
	configDS "github.com/stackrox/rox/central/config/datastore/mocks"
	deploymentDS "github.com/stackrox/rox/central/deployment/datastore/mocks"
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
		pc := &storage.PrivateConfig{}
		pc.ClearMetrics()
		cds.EXPECT().GetPrivateConfig(gomock.Any()).Times(1).Return(pc,
			nil)
		runner := makeRunner(&runnerDatastores{})
		runner.initialize(cds)
		assert.NotNil(t, runner)

		ctx := context.Background()
		assert.NotPanics(t, func() {
			runner[0].Gather(ctx)
		})

		cfg, err := runner.ValidateConfiguration(nil)
		assert.NoError(t, err)
		assert.NotNil(t, cfg)
		runner.Reconfigure(RunnerConfiguration{})

		assert.NotPanics(t, func() {
			runner[0].Gather(ctx)
		})
	})

	t.Run("err configuration", func(t *testing.T) {
		cds := configDS.NewMockDataStore(ctrl)
		cds.EXPECT().GetPrivateConfig(gomock.Any()).Times(1).Return(
			nil,
			errors.New("DB error"))
		runner := makeRunner(&runnerDatastores{})
		assert.NotNil(t, runner)
		runner.initialize(cds)

		ctx := context.Background()
		assert.NotPanics(t, func() {
			runner[0].Gather(ctx)
		})

		cfg, err := runner.ValidateConfiguration(nil)
		assert.NoError(t, err)
		assert.NotNil(t, cfg)
		runner.Reconfigure(RunnerConfiguration{})

		assert.NotPanics(t, func() {
			runner[0].Gather(ctx)
		})
	})
}

func TestRunner_ServeHTTP(t *testing.T) {
	ctrl := gomock.NewController(t)

	cds := configDS.NewMockDataStore(ctrl)

	cds.EXPECT().GetPrivateConfig(gomock.Any()).Times(1).Return(
		storage.PrivateConfig_builder{
			Metrics: storage.PrometheusMetrics_builder{
				ImageVulnerabilities: storage.PrometheusMetrics_Group_builder{
					GatheringPeriodMinutes: 10,
					Descriptors: map[string]*storage.PrometheusMetrics_Group_Labels{
						"metric1": storage.PrometheusMetrics_Group_Labels_builder{
							Labels: []string{"Cluster", "Severity"},
						}.Build(),
					}}.Build(),
				PolicyViolations: storage.PrometheusMetrics_Group_builder{
					GatheringPeriodMinutes: 10,
					Descriptors: map[string]*storage.PrometheusMetrics_Group_Labels{
						"metric2": storage.PrometheusMetrics_Group_Labels_builder{
							Labels: []string{"Cluster", "Policy", "Categories"},
						}.Build(),
					}}.Build()}.Build()}.Build(),
		nil)

	dds := deploymentDS.NewMockDataStore(ctrl)

	dds.EXPECT().WalkByQuery(gomock.Any(), gomock.Any(), gomock.Any()).
		Times(1).
		Do(func(_ context.Context, _ *v1.Query, f func(*storage.Deployment) error) {
			deployment := &storage.Deployment{}
			deployment.SetName("deployment1")
			deployment.SetClusterName("cluster1")
			_ = f(deployment)
		}).
		Return(nil)

	dds.EXPECT().GetImagesForDeployment(gomock.Any(), gomock.Any()).
		Times(1).Return([]*storage.Image{storage.Image_builder{
		Names: []*storage.ImageName{storage.ImageName_builder{FullName: "fullname1"}.Build()},
		Scan: storage.ImageScan_builder{
			Components: []*storage.EmbeddedImageScanComponent{storage.EmbeddedImageScanComponent_builder{
				Vulns: []*storage.EmbeddedVulnerability{storage.EmbeddedVulnerability_builder{
					Severity: storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY,
				}.Build()},
			}.Build()},
		}.Build()}.Build(),
	}, nil)

	ads := alertDS.NewMockDataStore(ctrl)

	ads.EXPECT().WalkByQuery(gomock.Any(), gomock.Any(), gomock.Any()).
		Times(1).
		Do(func(_ context.Context, _ *v1.Query, f func(*storage.Alert) error) {
			av := &storage.Alert_Violation{}
			av.SetMessage("violation")
			policy := &storage.Policy{}
			policy.SetName("Test Policy")
			policy.SetCategories([]string{"catB", "catA"})
			alert := &storage.Alert{}
			alert.SetClusterName("cluster1")
			alert.SetViolations([]*storage.Alert_Violation{
				av,
			})
			alert.SetPolicy(policy)
			_ = f(alert)
		}).
		Return(nil)

	runner := makeRunner(&runnerDatastores{deployments: dds, alerts: ads})
	runner.initialize(cds)
	runner[0].Gather(makeAdminContext(t))
	runner[1].Gather(makeAdminContext(t))

	expectedBody := func(metricName, decription, labels, vector string) string {
		metricName = "rox_central_" + metricName
		return fmt.Sprintf("# HELP %s The total number of %s aggregated by %s and gathered every 10m0s\n"+
			"# TYPE %s gauge\n%s{%s} 1\n", metricName, decription, labels, metricName, metricName, vector)
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
		assert.Contains(t, string(body),
			expectedBody("image_vuln_metric1", "image vulnerabilities",
				"Cluster,Severity",
				`Cluster="cluster1",Severity="IMPORTANT_VULNERABILITY_SEVERITY"`))
		assert.Contains(t, string(body),
			expectedBody("policy_violation_metric2", "policy violations",
				"Cluster,Policy,Categories",
				`Categories="catA,catB",Cluster="cluster1",Policy="Test Policy"`))
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
