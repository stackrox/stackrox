package deployment

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stackrox/rox/central/processindicator/views"
	"github.com/stackrox/rox/central/risk/getters"
	imageMultiplier "github.com/stackrox/rox/central/risk/multipliers/image"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClassifyCuratedRiskWorkloads(t *testing.T) {
	ctx := context.Background()

	workloads := []struct {
		name           string
		deployment     *storage.Deployment
		imageRisk      *storage.Risk
		alerts         []*storage.ListAlert
		processes      []*views.ProcessIndicatorRiskView
		expected       storage.RiskPriorityClassification
		expectedReason string
	}{
		{
			name: "benign-noisy-backlog",
			deployment: deploymentWithExposures("benign-noisy-backlog",
				storage.PortConfig_INTERNAL,
				storage.PortConfig_INTERNAL,
				storage.PortConfig_INTERNAL,
				storage.PortConfig_INTERNAL,
				storage.PortConfig_INTERNAL,
				storage.PortConfig_INTERNAL,
				storage.PortConfig_INTERNAL,
				storage.PortConfig_INTERNAL,
				storage.PortConfig_INTERNAL,
				storage.PortConfig_INTERNAL,
			),
			imageRisk: imageRiskWithSummaryAndResults(
				"benign-noisy-backlog-image",
				&storage.RiskVulnerabilitySummary{ModerateCount: 34, MaxSeverity: storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY},
				fixedRiskResult(imageMultiplier.VulnerabilitiesHeading, 4.0),
				fixedRiskResult(imageMultiplier.ImageAgeHeading, 1.5),
				fixedRiskResult(imageMultiplier.ComponentCountHeading, 1.5),
				fixedRiskResult(imageMultiplier.RiskyComponentCountHeading, 1.5),
			),
			alerts: []*storage.ListAlert{
				activePolicyAlertForClassification("Latest tag used", storage.Severity_LOW_SEVERITY),
				activePolicyAlertForClassification("No resource requests or limits", storage.Severity_LOW_SEVERITY),
				activePolicyAlertForClassification("Missing owner label", storage.Severity_LOW_SEVERITY),
			},
			expected:       storage.RiskPriorityClassification_RISK_PRIORITY_IMPROVE_POSTURE,
			expectedReason: "posture-only score boosters should not become urgent risk",
		},
		{
			name:       "public-critical-rce-with-credential",
			deployment: deploymentWithSecretForClassification(deploymentWithExposures("public-critical-rce-with-credential", storage.PortConfig_EXTERNAL), "prod-cloud-credentials"),
			imageRisk: imageRiskWithSummaryAndResults(
				"public-critical-rce-with-credential-image",
				&storage.RiskVulnerabilitySummary{CriticalCount: 1, MaxSeverity: storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY, NetworkAttackVector: true, CisaKev: true, MaxEpssPercentile: 0.99},
				fixedRiskResult(imageMultiplier.VulnerabilitiesHeading, 1.27075),
			),
			expected:       storage.RiskPriorityClassification_RISK_PRIORITY_REMEDIATE_NOW,
			expectedReason: "external exposure plus critical/known-exploited vulnerability is immediate remediation",
		},
		{
			name:       "critical-runtime-alert-and-baseline-break",
			deployment: deploymentWithExposures("critical-runtime-alert-and-baseline-break", storage.PortConfig_EXTERNAL),
			imageRisk: imageRiskWithSummaryAndResults(
				"critical-runtime-alert-and-baseline-break-image",
				&storage.RiskVulnerabilitySummary{CriticalCount: 1, MaxSeverity: storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY},
				fixedRiskResult(imageMultiplier.VulnerabilitiesHeading, 1.27075),
			),
			alerts: []*storage.ListAlert{
				activePolicyAlertForClassification("Cryptocurrency Mining Process Execution", storage.Severity_CRITICAL_SEVERITY),
			},
			processes:      suspiciousProcessesForClassification("sh", "curl"),
			expected:       storage.RiskPriorityClassification_RISK_PRIORITY_RESPOND_NOW,
			expectedReason: "critical active policy alerts are MVP response cases",
		},
		{
			name:           "unexpected-process",
			deployment:     deploymentWithExposures("unexpected-process", storage.PortConfig_INTERNAL),
			processes:      suspiciousProcessesForClassification("sh"),
			expected:       storage.RiskPriorityClassification_RISK_PRIORITY_INVESTIGATE_NOW,
			expectedReason: "process baseline deviations are investigation cases",
		},
		{
			name:           "cluster-wide-secret-reader",
			deployment:     clusterAdminDeploymentForClassification(deploymentWithExposures("cluster-wide-secret-reader", storage.PortConfig_INTERNAL)),
			expected:       storage.RiskPriorityClassification_RISK_PRIORITY_REMEDIATE,
			expectedReason: "cluster-wide identity blast radius is material remediation even without an external path",
		},
		{
			name:       "public-privileged-critical-rce",
			deployment: privilegedDeploymentWithSecretForClassification(deploymentWithExposures("public-privileged-critical-rce", storage.PortConfig_EXTERNAL), "prod-cloud-credentials"),
			imageRisk: imageRiskWithSummaryAndResults(
				"public-privileged-critical-rce-image",
				&storage.RiskVulnerabilitySummary{CriticalCount: 1, MaxSeverity: storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY},
				fixedRiskResult(imageMultiplier.VulnerabilitiesHeading, 1.27075),
			),
			expected:       storage.RiskPriorityClassification_RISK_PRIORITY_REMEDIATE_NOW,
			expectedReason: "public privileged critical-vulnerability workload is immediate remediation",
		},
	}

	for _, workload := range workloads {
		t.Run(workload.name, func(t *testing.T) {
			scorer := NewDeploymentScorer(getters.MockAlertsSearcher{Alerts: workload.alerts}, fakeProcessBaselineEvaluatorForClassification{processes: workload.processes})
			var imageRisks []*storage.Risk
			if workload.imageRisk != nil {
				imageRisks = append(imageRisks, workload.imageRisk)
			}
			risk := scorer.Score(ctx, workload.deployment, imageRisks)
			require.NotNil(t, risk)
			assert.Equal(t, workload.expected, risk.GetPriorityClassification(), workload.expectedReason)
		})
	}
}

func TestClassifyCuratedRiskWorkloadsOrdersByClass(t *testing.T) {
	classes := []storage.RiskPriorityClassification{
		storage.RiskPriorityClassification_RISK_PRIORITY_RESPOND_NOW,
		storage.RiskPriorityClassification_RISK_PRIORITY_INVESTIGATE_NOW,
		storage.RiskPriorityClassification_RISK_PRIORITY_REMEDIATE_NOW,
		storage.RiskPriorityClassification_RISK_PRIORITY_REMEDIATE,
		storage.RiskPriorityClassification_RISK_PRIORITY_IMPROVE_POSTURE,
	}
	for i := 1; i < len(classes); i++ {
		assert.Less(t, int32(classes[i-1]), int32(classes[i]))
	}
}

func deploymentWithExposures(id string, exposures ...storage.PortConfig_ExposureLevel) *storage.Deployment {
	ports := make([]*storage.PortConfig, 0, len(exposures))
	for i, exposure := range exposures {
		ports = append(ports, &storage.PortConfig{
			Name:          fmt.Sprintf("port-%d", i),
			ContainerPort: int32(8000 + i),
			Exposure:      exposure,
		})
	}
	return &storage.Deployment{Id: id, Name: id, ClusterId: "demo-cluster", Namespace: "risk-demo", Ports: ports}
}

func imageRiskWithSummaryAndResults(id string, summary *storage.RiskVulnerabilitySummary, results ...*storage.Risk_Result) *storage.Risk {
	var score float32 = 1
	for _, result := range results {
		score *= result.GetScore()
	}
	return &storage.Risk{Id: id, Score: score, Results: results, VulnerabilitySummary: summary}
}

func fixedRiskResult(name string, score float32) *storage.Risk_Result {
	return &storage.Risk_Result{Name: name, Score: score, Factors: []*storage.Risk_Result_Factor{{Message: name}}}
}

func activePolicyAlertForClassification(name string, severity storage.Severity) *storage.ListAlert {
	return &storage.ListAlert{
		Id:    name,
		State: storage.ViolationState_ACTIVE,
		Policy: &storage.ListAlertPolicy{
			Name:     name,
			Severity: severity,
		},
	}
}

func deploymentWithSecretForClassification(deployment *storage.Deployment, secretName string) *storage.Deployment {
	deployment.Containers = []*storage.Container{{Name: "app", SecurityContext: &storage.SecurityContext{DropCapabilities: []string{"ALL"}}, Secrets: []*storage.EmbeddedSecret{{Name: secretName, Path: "/var/run/secrets/demo"}}}}
	return deployment
}

func privilegedDeploymentWithSecretForClassification(deployment *storage.Deployment, secretName string) *storage.Deployment {
	deployment.Containers = []*storage.Container{{Name: "app", SecurityContext: &storage.SecurityContext{Privileged: true, AddCapabilities: []string{"SYS_ADMIN"}, DropCapabilities: []string{"ALL"}}, Secrets: []*storage.EmbeddedSecret{{Name: secretName, Path: "/var/run/secrets/demo"}}}}
	return deployment
}

func clusterAdminDeploymentForClassification(deployment *storage.Deployment) *storage.Deployment {
	deployment.ServiceAccount = "cluster-admin-workload"
	deployment.ServiceAccountPermissionLevel = storage.PermissionLevel_CLUSTER_ADMIN
	deployment.AutomountServiceAccountToken = true
	return deployment
}

func suspiciousProcessesForClassification(names ...string) []*views.ProcessIndicatorRiskView {
	processes := make([]*views.ProcessIndicatorRiskView, 0, len(names))
	for _, name := range names {
		processes = append(processes, &views.ProcessIndicatorRiskView{SignalName: name, SignalArgs: strings.TrimSpace("demo"), ContainerName: "app"})
	}
	return processes
}

type fakeProcessBaselineEvaluatorForClassification struct {
	processes []*views.ProcessIndicatorRiskView
}

func (f fakeProcessBaselineEvaluatorForClassification) EvaluateBaselinesAndPersistResult(_ *storage.Deployment) ([]*views.ProcessIndicatorRiskView, error) {
	return f.processes, nil
}
