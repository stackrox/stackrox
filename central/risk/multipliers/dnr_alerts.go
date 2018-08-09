package multipliers

import (
	"fmt"
	"math"
	"sort"

	"github.com/stackrox/rox/central/dnrintegration"
	"github.com/stackrox/rox/central/risk/getters"
	"github.com/stackrox/rox/generated/api/v1"
)

const (
	// DnrAlertsHeading is the risk result name for scores calculated by this multiplier.
	DnrAlertsHeading = "Runtime Alerts"

	dnrAlertsSaturation = float32(300)
)

// dnrAlertMultiplier is a scorer that uses alerts from Detect & Respond.
type dnrAlertMultiplier struct {
	integrationGetter getters.DNRIntegrationGetter
}

// NewDNRAlert provides a Multiplier using DNR alerts as a risk factor.
func NewDNRAlert(integrationGetter getters.DNRIntegrationGetter) Multiplier {
	return &dnrAlertMultiplier{
		integrationGetter: integrationGetter,
	}
}

// Score takes a deployment and evaluates its risk based on the service configuration
func (d *dnrAlertMultiplier) Score(deployment *v1.Deployment) *v1.Risk_Result {
	integration, exists, err := d.integrationGetter.ForCluster(deployment.GetClusterId())
	if err != nil {
		logger.Errorf("Error retrieving D&R integration for cluster %s: %s", deployment.GetClusterId(), err)
		return nil
	}
	if !exists {
		return nil
	}
	alerts, err := integration.Alerts(deployment.GetNamespace(), deployment.GetName())
	if err != nil {
		logger.Errorf("Couldn't get D&R alerts for deployment %#v: %s", deployment, err)
	}

	factors, severity := d.computeSeverityAndFactors(alerts)
	if severity == 0 {
		return nil
	}
	return &v1.Risk_Result{
		Name:    DnrAlertsHeading,
		Factors: factors,
		Score:   severity,
	}
}

// Compute severity based on the score returned by D&R.
// For reference, the mapping used over there is:
// Low:      25,
// Medium:   50,
// High:     75,
// Critical: 100,
func (d *dnrAlertMultiplier) computeSeverityAndFactors(alerts []dnrintegration.PolicyAlert) (factors []string, severity float32) {
	// alertWithCount represents an alert with the number of violations of it we observed.
	type alertWithCount struct {
		dnrintegration.PolicyAlert
		violationsCount int
	}
	// getSeverityScore computes the severity score of an alert.
	// We currently weight severity as log_2(1 + count).
	// Some sample values to understand how this grows:
	// Count  Log
	// 2      1.0
	// 3      1.5849625007211563
	// 4      2.0
	// 5      2.321928094887362
	// 6      2.584962500721156
	// 7      2.807354922057604
	// 8      3.0
	// 9      3.1699250014423126
	// 10     3.3219280948873626
	// 15     3.9068905956085187
	// 20     4.321928094887363
	// 30     4.906890595608519
	// 40     5.321928094887363
	// 50     5.643856189774724
	// 70     6.129283016944967
	// 90     6.491853096329675
	// 100    6.643856189774725
	getSeverityScore := func(alert alertWithCount) float64 {
		return alert.SeverityScore * math.Log1p(float64(alert.violationsCount)) / math.Ln2
	}

	alertsWithCountsMap := make(map[string]*alertWithCount)
	for _, alert := range alerts {
		if _, exists := alertsWithCountsMap[alert.PolicyName]; !exists {
			alertsWithCountsMap[alert.PolicyName] = &alertWithCount{
				PolicyAlert: alert,
			}
		}
		alertsWithCountsMap[alert.PolicyName].violationsCount++
	}

	alertsWithCounts := make([]alertWithCount, 0, len(alertsWithCountsMap))
	for _, a := range alertsWithCountsMap {
		alertsWithCounts = append(alertsWithCounts, *a)
	}

	// Sort alerts in descending order of severity score.
	sort.SliceStable(alertsWithCounts, func(i, j int) bool {
		return getSeverityScore(alertsWithCounts[i]) > getSeverityScore(alertsWithCounts[j])
	})

	const maxLines = 5
	for i, alert := range alertsWithCounts {
		if i == maxLines {
			break
		}
		severity += float32(getSeverityScore(alert))

		// If it occurred more than once, print the number of occurrences in the factor string.
		// However, if it occurs more than maxCount times, just print maxCount+ instead of the exact number.
		const maxCount = 10
		var countString string
		if alert.violationsCount > 1 {
			if alert.violationsCount < maxCount {
				countString = fmt.Sprintf(" (%dx)", alert.violationsCount)
			} else {
				countString = fmt.Sprintf(" (%d+ x)", maxCount)
			}
		}
		factors = append(factors, fmt.Sprintf("%s (Severity: %s)%s", alert.PolicyName, alert.SeverityWord, countString))
	}

	if severity == 0 {
		return
	}
	if severity > dnrAlertsSaturation {
		severity = dnrAlertsSaturation
	}
	severity = 1 + (severity / dnrAlertsSaturation)

	// If we have more than `maxLines` lines, summarize the rest of the alerts in one more line.
	if len(alertsWithCounts) > maxLines {
		remainingAlerts := len(alertsWithCounts) - maxLines
		factors = append(factors, fmt.Sprintf("%d Other Alerts", remainingAlerts))
	}
	return
}
