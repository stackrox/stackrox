//go:build risk_demo

package deployment

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/stackrox/rox/central/processindicator/views"
	"github.com/stackrox/rox/central/risk/getters"
	deploymentMultiplier "github.com/stackrox/rox/central/risk/multipliers/deployment"
	imageMultiplier "github.com/stackrox/rox/central/risk/multipliers/image"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests intentionally characterize unintuitive behavior in the current ACS Risk formula.
// They are behind the risk_demo build tag so they can be used as readable proof-of-concept demos
// without permanently enshrining the current behavior as desired product behavior.
//
// Run with:
//
//   go test -tags risk_demo ./central/risk/scorer/deployment -run TestCurrentRiskModelDemo -v

func TestCurrentRiskModelDemo_InternalPortCountCanMaxReachability(t *testing.T) {
	ctx := context.Background()
	reachability := deploymentMultiplier.NewReachability()

	oneExternalPort := deploymentWithPorts("one-external-port", []storage.PortConfig_ExposureLevel{
		storage.PortConfig_EXTERNAL,
	})
	tenInternalPorts := deploymentWithPorts("ten-internal-ports", []storage.PortConfig_ExposureLevel{
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
	})

	externalResult := reachability.Score(ctx, oneExternalPort, nil)
	internalResult := reachability.Score(ctx, tenInternalPorts, nil)
	require.NotNil(t, externalResult)
	require.NotNil(t, internalResult)

	t.Logf("one EXTERNAL port: reachability multiplier = %.2fx", externalResult.GetScore())
	t.Logf("ten INTERNAL ports: reachability multiplier = %.2fx", internalResult.GetScore())
	t.Logf("demo point: reachability sums declared ports, so enough internal ports can max the multiplier even without proving vulnerable code, a live listener, observed traffic, or public reachability")

	assert.InDelta(t, 1.3, externalResult.GetScore(), 0.0001)
	assert.InDelta(t, 2.0, internalResult.GetScore(), 0.0001)
	assert.Greater(t, internalResult.GetScore(), externalResult.GetScore())
}

func TestCurrentRiskModelDemo_LowOrModerateCVEVolumeCanOutweighSingleCriticalExposure(t *testing.T) {
	ctx := context.Background()

	criticalVulnRisk := imageVulnerabilityRiskFromCounts(t, "single-critical-rce", vulnCounts{
		critical: 1,
	})
	moderateVolumeRisk := imageVulnerabilityRiskFromCounts(t, "moderate-volume-backlog", vulnCounts{
		moderate: 34,
	})
	lowVolumeRisk := imageVulnerabilityRiskFromCounts(t, "low-volume-backlog", vulnCounts{
		low: 251,
	})

	t.Logf("image with 1 Critical CVE: Image Vulnerabilities multiplier = %.4fx", riskResultScore(t, criticalVulnRisk, imageMultiplier.VulnerabilitiesHeading))
	t.Logf("image with 34 Moderate CVEs: Image Vulnerabilities multiplier = %.4fx", riskResultScore(t, moderateVolumeRisk, imageMultiplier.VulnerabilitiesHeading))
	t.Logf("image with 251 Low CVEs: Image Vulnerabilities multiplier = %.4fx", riskResultScore(t, lowVolumeRisk, imageMultiplier.VulnerabilitiesHeading))
	t.Logf("demo point: Scanner is not involved; these images are synthetic scan results fed directly into the current image vulnerability multiplier")

	scorer := &deploymentScorerImpl{ConfiguredMultipliers: []deploymentMultiplier.Multiplier{
		deploymentMultiplier.NewImageMultiplier(imageMultiplier.VulnerabilitiesHeading),
		deploymentMultiplier.NewReachability(),
	}}

	externalSingleCritical := deploymentWithPorts("external-single-critical", []storage.PortConfig_ExposureLevel{
		storage.PortConfig_EXTERNAL,
	})
	internalModerateVolume := deploymentWithPorts("internal-moderate-volume", []storage.PortConfig_ExposureLevel{
		storage.PortConfig_INTERNAL,
	})
	internalLowVolume := deploymentWithPorts("internal-low-volume", []storage.PortConfig_ExposureLevel{
		storage.PortConfig_INTERNAL,
	})

	externalRisk := scorer.Score(ctx, externalSingleCritical, []*storage.Risk{criticalVulnRisk})
	internalModerateRisk := scorer.Score(ctx, internalModerateVolume, []*storage.Risk{moderateVolumeRisk})
	internalLowRisk := scorer.Score(ctx, internalLowVolume, []*storage.Risk{lowVolumeRisk})
	require.NotNil(t, externalRisk)
	require.NotNil(t, internalModerateRisk)
	require.NotNil(t, internalLowRisk)

	t.Logf("deployment score: EXTERNAL workload with 1 Critical CVE = %.4f", externalRisk.GetScore())
	t.Logf("deployment score: INTERNAL workload with 34 Moderate CVEs = %.4f", internalModerateRisk.GetScore())
	t.Logf("deployment score: INTERNAL workload with 251 Low CVEs = %.4f", internalLowRisk.GetScore())
	t.Logf("demo point: enough low/moderate vulnerability volume saturates the image vulnerability multiplier and can beat a single critical CVE even when the critical workload has external exposure")

	assert.Greater(t, riskResultScore(t, moderateVolumeRisk, imageMultiplier.VulnerabilitiesHeading), riskResultScore(t, criticalVulnRisk, imageMultiplier.VulnerabilitiesHeading))
	assert.Greater(t, riskResultScore(t, lowVolumeRisk, imageMultiplier.VulnerabilitiesHeading), riskResultScore(t, criticalVulnRisk, imageMultiplier.VulnerabilitiesHeading))
	assert.Greater(t, internalModerateRisk.GetScore(), externalRisk.GetScore())
	assert.Greater(t, internalLowRisk.GetScore(), externalRisk.GetScore())
}

func TestCurrentRiskModelDemo_ServiceAccountBlastRadiusDoesNotAffectScore(t *testing.T) {
	ctx := context.Background()
	scorer := &deploymentScorerImpl{ConfiguredMultipliers: []deploymentMultiplier.Multiplier{
		deploymentMultiplier.NewReachability(),
	}}

	baseline := deploymentWithPorts("baseline", []storage.PortConfig_ExposureLevel{storage.PortConfig_INTERNAL})
	clusterAdmin := deploymentWithPorts("cluster-admin", []storage.PortConfig_ExposureLevel{storage.PortConfig_INTERNAL})
	clusterAdmin.ServiceAccount = "cluster-admin-workload"
	clusterAdmin.ServiceAccountPermissionLevel = storage.PermissionLevel_CLUSTER_ADMIN
	clusterAdmin.AutomountServiceAccountToken = true

	baselineRisk := scorer.Score(ctx, baseline, nil)
	clusterAdminRisk := scorer.Score(ctx, clusterAdmin, nil)
	require.NotNil(t, baselineRisk)
	require.NotNil(t, clusterAdminRisk)

	t.Logf("baseline internal workload score = %.4f", baselineRisk.GetScore())
	t.Logf("otherwise identical workload with CLUSTER_ADMIN service account and automounted token score = %.4f", clusterAdminRisk.GetScore())
	t.Logf("demo point: service account permission level and automounted token exist on storage.Deployment, but this scorer does not use them as direct risk factors")

	assert.Equal(t, baselineRisk.GetScore(), clusterAdminRisk.GetScore())
}

type vulnCounts struct {
	critical  int
	important int
	moderate  int
	low       int
}

func imageVulnerabilityRiskFromCounts(t *testing.T, imageName string, counts vulnCounts) *storage.Risk {
	t.Helper()

	image := &storage.Image{
		Id: imageName,
		Name: &storage.ImageName{
			FullName: imageName,
		},
		Scan: &storage.ImageScan{
			Components: []*storage.EmbeddedImageScanComponent{
				{
					Name:    "demo-component",
					Version: "1.0.0",
					Vulns:   vulnerabilitiesFromCounts(counts),
				},
			},
		},
	}

	result := imageMultiplier.NewVulnerabilities().Score(context.Background(), image)
	require.NotNil(t, result)

	return &storage.Risk{
		Score: result.GetScore(),
		Results: []*storage.Risk_Result{
			result,
		},
	}
}

func vulnerabilitiesFromCounts(counts vulnCounts) []*storage.EmbeddedVulnerability {
	vulns := make([]*storage.EmbeddedVulnerability, 0, counts.critical+counts.important+counts.moderate+counts.low)
	appendSeverity := func(prefix string, count int, severity storage.VulnerabilitySeverity) {
		for i := 0; i < count; i++ {
			vulns = append(vulns, &storage.EmbeddedVulnerability{
				Cve:          fmt.Sprintf("CVE-2026-%s-%04d", strings.ToUpper(prefix), i),
				ScoreVersion: storage.EmbeddedVulnerability_V3,
				Severity:     severity,
			})
		}
	}

	appendSeverity("critical", counts.critical, storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY)
	appendSeverity("important", counts.important, storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY)
	appendSeverity("moderate", counts.moderate, storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY)
	appendSeverity("low", counts.low, storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY)

	return vulns
}

func deploymentWithPorts(id string, exposures []storage.PortConfig_ExposureLevel) *storage.Deployment {
	ports := make([]*storage.PortConfig, 0, len(exposures))
	for i, exposure := range exposures {
		ports = append(ports, &storage.PortConfig{
			Name:          fmt.Sprintf("port-%d", i),
			ContainerPort: int32(8000 + i),
			Exposure:      exposure,
		})
	}

	return &storage.Deployment{
		Id:        id,
		Name:      id,
		ClusterId: "demo-cluster",
		Namespace: "risk-demo",
		Ports:     ports,
	}
}

func TestCurrentRiskModelDemo_DangerousCasesCanBeBuriedByBenignScoreBoosters(t *testing.T) {
	ctx := context.Background()

	type demoWorkload struct {
		name        string
		description string
		deployment  *storage.Deployment
		imageRisk   *storage.Risk
		alerts      []*storage.ListAlert
		processes   []*views.ProcessIndicatorRiskView
	}

	workloads := []demoWorkload{
		{
			name: "benign-noisy-backlog",
			description: strings.Join([]string{
				"many low/moderate CVEs",
				"old image",
				"many components",
				"attacker-useful utility packages",
				"ten internal declared ports",
				"several low-severity hygiene policy violations",
				"no asserted public exposure, exploit evidence, runtime behavior, or elevated identity",
			}, "; "),
			deployment: deploymentWithPorts("benign-noisy-backlog", []storage.PortConfig_ExposureLevel{
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
			}),
			imageRisk: imageRiskWithResults(
				"benign-noisy-backlog-image",
				imageVulnerabilityRiskFromCounts(t, "benign-noisy-backlog-image", vulnCounts{moderate: 34}).GetResults()[0],
				fixedImageRiskResult(imageMultiplier.ImageAgeHeading, 1.5, "Image is older than 270 days"),
				fixedImageRiskResult(imageMultiplier.ComponentCountHeading, 1.5, "Image has many components"),
				fixedImageRiskResult(imageMultiplier.RiskyComponentCountHeading, 1.5, "Image contains shell/curl/wget style utilities"),
			),
			alerts: []*storage.ListAlert{
				activePolicyAlert("Latest tag used", storage.Severity_LOW_SEVERITY),
				activePolicyAlert("No resource requests or limits", storage.Severity_LOW_SEVERITY),
				activePolicyAlert("Missing owner label", storage.Severity_LOW_SEVERITY),
				activePolicyAlert("Writable root filesystem", storage.Severity_LOW_SEVERITY),
				activePolicyAlert("Package manager present in image", storage.Severity_LOW_SEVERITY),
			},
		},
		{
			name: "public-kev-rce-with-credential",
			description: strings.Join([]string{
				"public external exposure",
				"one Critical network RCE CVE",
				"assume high EPSS / CISA KEV metadata exists but is not used by this scorer",
				"mounted credential Secret",
				"hardened otherwise",
			}, "; "),
			deployment: deploymentWithSecret(
				deploymentWithPorts("public-kev-rce-with-credential", []storage.PortConfig_ExposureLevel{storage.PortConfig_EXTERNAL}),
				"prod-cloud-credentials",
			),
			imageRisk: imageVulnerabilityRiskFromCounts(t, "public-kev-rce-with-credential-image", vulnCounts{critical: 1}),
		},
		{
			name: "critical-runtime-alert-and-baseline-break",
			description: strings.Join([]string{
				"public external exposure",
				"Critical runtime policy alert",
				"several processes running outside baseline",
				"one Critical CVE",
				"these are response/investigation signals, not posture backlog",
			}, "; "),
			deployment: deploymentWithPorts("critical-runtime-alert-and-baseline-break", []storage.PortConfig_ExposureLevel{storage.PortConfig_EXTERNAL}),
			imageRisk:  imageVulnerabilityRiskFromCounts(t, "critical-runtime-alert-and-baseline-break-image", vulnCounts{critical: 1}),
			alerts: []*storage.ListAlert{
				activePolicyAlert("Cryptocurrency Mining Process Execution", storage.Severity_CRITICAL_SEVERITY),
			},
			processes: suspiciousProcesses("sh", "curl", "nc"),
		},
		{
			name: "cluster-wide-secret-reader",
			description: strings.Join([]string{
				"service account has cluster-admin / cluster-wide Secret access",
				"automounted service account token",
				"internal port only",
				"identity blast radius is present on storage.Deployment but is not a direct scoring factor",
			}, "; "),
			deployment: clusterAdminDeployment(
				deploymentWithPorts("cluster-wide-secret-reader", []storage.PortConfig_ExposureLevel{storage.PortConfig_INTERNAL}),
			),
		},
		{
			name: "public-privileged-critical-rce",
			description: strings.Join([]string{
				"public external exposure",
				"privileged container with SYS_ADMIN",
				"mounted credential Secret",
				"one Critical CVE",
				"powerful impact potential, but still far below noisy backlog in current formula",
			}, "; "),
			deployment: privilegedDeploymentWithSecret(
				deploymentWithPorts("public-privileged-critical-rce", []storage.PortConfig_ExposureLevel{storage.PortConfig_EXTERNAL}),
				"prod-cloud-credentials",
			),
			imageRisk: imageVulnerabilityRiskFromCounts(t, "public-privileged-critical-rce-image", vulnCounts{critical: 1}),
		},
	}

	type scoredWorkload struct {
		demoWorkload
		risk *storage.Risk
	}
	scored := make([]scoredWorkload, 0, len(workloads))
	for _, workload := range workloads {
		scorer := NewDeploymentScorer(getters.MockAlertsSearcher{Alerts: workload.alerts}, fakeProcessBaselineEvaluator{processes: workload.processes})
		var imageRisks []*storage.Risk
		if workload.imageRisk != nil {
			imageRisks = append(imageRisks, workload.imageRisk)
		}
		risk := scorer.Score(ctx, workload.deployment, imageRisks)
		require.NotNil(t, risk, workload.name)
		scored = append(scored, scoredWorkload{demoWorkload: workload, risk: risk})
	}

	sort.Slice(scored, func(i, j int) bool {
		return scored[i].risk.GetScore() > scored[j].risk.GetScore()
	})

	t.Log("current formula ranking, highest raw score first:")
	for i, workload := range scored {
		t.Logf("%d. %s score=%.4f -- %s", i+1, workload.name, workload.risk.GetScore(), workload.description)
		logRiskResultsInScorerOrder(t, workload.risk)
	}
	t.Log("demo point: the first row should be posture/backlog work in a case-based UX, but the current multiplicative score ranks it above response/remediation cases with public exposure, Critical CVEs, runtime alerts, suspicious processes, credentials, privilege, or cluster-wide identity blast radius")

	assert.Equal(t, "benign-noisy-backlog", scored[0].name)
	for _, workload := range scored[1:] {
		assert.Greater(t, scored[0].risk.GetScore(), workload.risk.GetScore(), "benign noisy backlog should bury %s under the current formula", workload.name)
	}
}

type multiplierDisplay struct {
	name     string
	maxScore float32
}

var deploymentMultiplierDisplayOrder = []multiplierDisplay{
	{name: deploymentMultiplier.PolicyViolationsHeading, maxScore: 4},
	{name: deploymentMultiplier.ProcessBaselineHeading, maxScore: 4},
	{name: imageMultiplier.VulnerabilitiesHeading, maxScore: 4},
	{name: deploymentMultiplier.ServiceConfigHeading, maxScore: 2},
	{name: deploymentMultiplier.ReachabilityHeading, maxScore: 2},
	{name: imageMultiplier.RiskyComponentCountHeading, maxScore: 1.5},
	{name: imageMultiplier.ComponentCountHeading, maxScore: 1.5},
	{name: imageMultiplier.ImageAgeHeading, maxScore: 1.5},
}

func logRiskResultsInScorerOrder(t *testing.T, risk *storage.Risk) {
	t.Helper()
	resultsByName := make(map[string]*storage.Risk_Result, len(risk.GetResults()))
	for _, result := range risk.GetResults() {
		resultsByName[result.GetName()] = result
	}
	for _, display := range deploymentMultiplierDisplayOrder {
		if result := resultsByName[display.name]; result != nil {
			t.Logf("   - %-36s (max %3.1fx): %7.4fx", display.name, display.maxScore, result.GetScore())
			continue
		}
		t.Logf("   - %-36s (max %3.1fx): %7.4fx", display.name, display.maxScore, float32(1))
	}
}

func riskResultScore(t *testing.T, risk *storage.Risk, name string) float32 {
	t.Helper()
	for _, result := range risk.GetResults() {
		if result.GetName() == name {
			return result.GetScore()
		}
	}
	t.Fatalf("risk result %q not found", name)
	return 0
}

func fixedImageRiskResult(name string, score float32, factor string) *storage.Risk_Result {
	return &storage.Risk_Result{
		Name:  name,
		Score: score,
		Factors: []*storage.Risk_Result_Factor{
			{Message: factor},
		},
	}
}

func imageRiskWithResults(id string, results ...*storage.Risk_Result) *storage.Risk {
	var score float32 = 1
	for _, result := range results {
		score *= result.GetScore()
	}
	return &storage.Risk{
		Id:      id,
		Score:   score,
		Results: results,
	}
}

func deploymentWithSecret(deployment *storage.Deployment, secretName string) *storage.Deployment {
	deployment.Containers = []*storage.Container{
		{
			Name: "app",
			SecurityContext: &storage.SecurityContext{
				DropCapabilities: []string{"ALL"},
			},
			Secrets: []*storage.EmbeddedSecret{
				{Name: secretName, Path: "/var/run/secrets/demo"},
			},
		},
	}
	return deployment
}

func privilegedDeploymentWithSecret(deployment *storage.Deployment, secretName string) *storage.Deployment {
	deployment.Containers = []*storage.Container{
		{
			Name: "app",
			SecurityContext: &storage.SecurityContext{
				Privileged:       true,
				AddCapabilities:  []string{"SYS_ADMIN"},
				DropCapabilities: []string{"ALL"},
			},
			Secrets: []*storage.EmbeddedSecret{
				{Name: secretName, Path: "/var/run/secrets/demo"},
			},
		},
	}
	return deployment
}

func clusterAdminDeployment(deployment *storage.Deployment) *storage.Deployment {
	deployment.ServiceAccount = "cluster-admin-workload"
	deployment.ServiceAccountPermissionLevel = storage.PermissionLevel_CLUSTER_ADMIN
	deployment.AutomountServiceAccountToken = true
	return deployment
}

func activePolicyAlert(name string, severity storage.Severity) *storage.ListAlert {
	return &storage.ListAlert{
		Id:    name,
		State: storage.ViolationState_ACTIVE,
		Policy: &storage.ListAlertPolicy{
			Name:     name,
			Severity: severity,
		},
	}
}

func suspiciousProcesses(names ...string) []*views.ProcessIndicatorRiskView {
	processes := make([]*views.ProcessIndicatorRiskView, 0, len(names))
	for _, name := range names {
		processes = append(processes, &views.ProcessIndicatorRiskView{
			SignalName:    name,
			SignalArgs:    "demo",
			ContainerName: "app",
		})
	}
	return processes
}

type fakeProcessBaselineEvaluator struct {
	processes []*views.ProcessIndicatorRiskView
}

func (f fakeProcessBaselineEvaluator) EvaluateBaselinesAndPersistResult(_ *storage.Deployment) ([]*views.ProcessIndicatorRiskView, error) {
	return f.processes, nil
}
