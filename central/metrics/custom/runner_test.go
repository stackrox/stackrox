package custom

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pkg/errors"
	alertDS "github.com/stackrox/rox/central/alert/datastore/mocks"
	configDS "github.com/stackrox/rox/central/config/datastore/mocks"
	deploymentDS "github.com/stackrox/rox/central/deployment/datastore/mocks"
	"github.com/stackrox/rox/central/metrics"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/authn/basic"
	identityMocks "github.com/stackrox/rox/pkg/grpc/authn/mocks"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		&storage.PrivateConfig{
			Metrics: &storage.PrometheusMetrics{
				ImageVulnerabilities: &storage.PrometheusMetrics_Group{
					GatheringPeriodMinutes: 10,
					Descriptors: map[string]*storage.PrometheusMetrics_Group_Labels{
						"metric1": {
							Labels: []string{"Cluster", "Severity"},
						},
					}},
				PolicyViolations: &storage.PrometheusMetrics_Group{
					GatheringPeriodMinutes: 10,
					Descriptors: map[string]*storage.PrometheusMetrics_Group_Labels{
						"metric2": {
							Labels: []string{"Cluster", "Policy", "Categories"},
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

	ads := alertDS.NewMockDataStore(ctrl)

	ads.EXPECT().WalkByQuery(gomock.Any(), gomock.Any(), gomock.Any()).
		Times(1).
		Do(func(_ context.Context, _ *v1.Query, f func(*storage.Alert) error) {
			_ = f(&storage.Alert{
				ClusterName: "cluster1",
				Violations: []*storage.Alert_Violation{
					{
						Message: "violation",
					},
				},
				Policy: &storage.Policy{
					Name:       "Test Policy",
					Categories: []string{"catB", "catA"},
				},
			})
		}).
		Return(nil)

	runner := makeRunner(&runnerDatastores{deployments: dds, alerts: ads})
	runner.initialize(cds)
	runner[0].Gather(makeAdminContext(t))
	runner[1].Gather(makeAdminContext(t))

	expectedBody := func(metricName, decription, labels, vector string) string {
		metricName = "rox_central_" + metricName
		return fmt.Sprintf("# HELP %s The total number of %s aggregated by %s, and gathered every 10m0s\n"+
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
				"Cluster, Severity",
				`Cluster="cluster1",Severity="IMPORTANT_VULNERABILITY_SEVERITY"`))
		assert.Contains(t, string(body),
			expectedBody("policy_violation_metric2", "policy violations",
				"Categories, Cluster, Policy",
				`Categories="catA,catB",Cluster="cluster1",Policy="Test Policy"`))
	})
}

func TestRunner_GlobalMetricsCheckCurrentPermissions(t *testing.T) {
	globalRegistry, err := metrics.GetGlobalRegistry()
	require.NoError(t, err)
	for _, metric := range []string{
		"admin_event_authz_test",
		"cfg_authz_test",
		"cert_exp_authz_test",
		"unknown_authz_test",
	} {
		require.NoError(t, globalRegistry.RegisterMetric(metric, metric, nil))
		globalRegistry.SetTotal(metric, nil, 1)
		t.Cleanup(func() { globalRegistry.UnregisterMetric(metric) })
	}

	runner := makeRunner(&runnerDatastores{})
	workflowIdentity := identityMocks.NewMockIdentity(gomock.NewController(t))
	workflowIdentity.EXPECT().UID().AnyTimes().Return("workflow-authz-test")
	workflowIdentity.EXPECT().Permissions().AnyTimes().Return(map[string]storage.Access{
		"WorkflowAdministration": storage.Access_READ_ACCESS,
	})
	workflowCtx := authn.ContextWithIdentity(context.Background(), workflowIdentity, t)
	t.Cleanup(func() { metrics.DeleteCustomRegistry("workflow-authz-test") })
	scopedIdentity := identityMocks.NewMockIdentity(gomock.NewController(t))
	scopedIdentity.EXPECT().UID().AnyTimes().Return("scoped-only-authz-test")
	scopedIdentity.EXPECT().Permissions().AnyTimes().Return(map[string]storage.Access{
		"Deployment": storage.Access_READ_ACCESS,
	})
	scopedCtx := authn.ContextWithIdentity(context.Background(), scopedIdentity, t)
	scopedRegistry, err := metrics.GetCustomRegistry("scoped-only-authz-test")
	require.NoError(t, err)
	const scopedMetric = "image_vuln_scoped_authz_test"
	require.NoError(t, scopedRegistry.RegisterMetric(scopedMetric, scopedMetric, nil))
	scopedRegistry.SetTotal(scopedMetric, nil, 1)
	t.Cleanup(func() { metrics.DeleteCustomRegistry("scoped-only-authz-test") })
	for _, test := range []struct {
		name          string
		ctx           context.Context
		allowedMetric string
		allowScoped   bool
	}{
		{"admin", makeAdminContext(t), "all", false},
		{"workflow only", workflowCtx, "cfg_authz_test", false},
		{"scoped only", scopedCtx, "", true},
		{"no access after admin", basic.ContextWithNoAccessIdentity(t, nil), "", false},
		{"anonymous", context.Background(), "", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			runner.ServeHTTP(rec, httptest.NewRequestWithContext(test.ctx, http.MethodGet, "/metrics", nil))
			require.Equal(t, http.StatusOK, rec.Code)
			for _, metric := range []string{"admin_event_authz_test", "cfg_authz_test", "cert_exp_authz_test"} {
				assert.Equal(t, test.allowedMetric == "all" || test.allowedMetric == metric,
					containsMetric(rec.Body.String(), metric), metric)
			}
			assert.False(t, containsMetric(rec.Body.String(), "unknown_authz_test"))
			assert.Equal(t, test.allowScoped, containsMetric(rec.Body.String(), scopedMetric))
		})
	}
}

func containsMetric(body, metric string) bool {
	return strings.Contains(body, "rox_central_"+metric+" 1\n")
}

func makeAdminContext(t *testing.T) context.Context {
	authProvider, _ := authproviders.NewProvider(
		authproviders.WithEnabled(true),
		authproviders.WithID(uuid.NewDummy().String()),
		authproviders.WithName("Test Auth Provider"),
	)
	return basic.ContextWithAdminIdentity(t, authProvider)
}
