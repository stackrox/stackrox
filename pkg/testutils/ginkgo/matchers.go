package ginkgo

import (
	"fmt"
	"time"

	"github.com/onsi/gomega/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/testutils/external"
)

// Custom Gomega matchers for StackRox-specific assertions

// HaveAlert matcher for checking alert presence
type HaveAlertMatcher struct {
	expectedCount   int
	actualCount     int
	expectedSeverity *storage.Severity
}

func HaveAlert() *HaveAlertMatcher {
	return &HaveAlertMatcher{expectedCount: 1}
}

func HaveAlerts(count int) *HaveAlertMatcher {
	return &HaveAlertMatcher{expectedCount: count}
}

func (m *HaveAlertMatcher) WithSeverity(severity storage.Severity) *HaveAlertMatcher {
	m.expectedSeverity = &severity
	return m
}

func (m *HaveAlertMatcher) Match(actual interface{}) (success bool, err error) {
	alerts, ok := actual.([]*storage.Alert)
	if !ok {
		return false, fmt.Errorf("HaveAlert matcher expects []*storage.Alert")
	}

	m.actualCount = len(alerts)

	// Check count
	if m.actualCount != m.expectedCount {
		return false, nil
	}

	// Check severity if specified
	if m.expectedSeverity != nil {
		for _, alert := range alerts {
			if alert.GetPolicy().GetSeverity() != *m.expectedSeverity {
				return false, nil
			}
		}
	}

	return true, nil
}

func (m *HaveAlertMatcher) FailureMessage(actual interface{}) (message string) {
	if m.expectedSeverity != nil {
		return fmt.Sprintf("Expected %d alerts with severity %s, but got %d alerts",
			m.expectedCount, m.expectedSeverity.String(), m.actualCount)
	}
	return fmt.Sprintf("Expected %d alerts, but got %d", m.expectedCount, m.actualCount)
}

func (m *HaveAlertMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	if m.expectedSeverity != nil {
		return fmt.Sprintf("Expected not to have %d alerts with severity %s, but got %d alerts",
			m.expectedCount, m.expectedSeverity.String(), m.actualCount)
	}
	return fmt.Sprintf("Expected not to have %d alerts, but got %d", m.expectedCount, m.actualCount)
}

// BeDeploymentBlocked matcher for checking if deployment is blocked
type BeDeploymentBlockedMatcher struct{}

func BeDeploymentBlocked() *BeDeploymentBlockedMatcher {
	return &BeDeploymentBlockedMatcher{}
}

func (m *BeDeploymentBlockedMatcher) Match(actual interface{}) (success bool, err error) {
	deploymentName, ok := actual.(string)
	if !ok {
		return false, fmt.Errorf("BeDeploymentBlocked matcher expects string (deployment name)")
	}

	// TODO: Implement actual deployment blocking check
	// This would check if the deployment exists and is in a blocked state
	_ = deploymentName
	return false, fmt.Errorf("deployment blocking check not implemented")
}

func (m *BeDeploymentBlockedMatcher) FailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf("Expected deployment %v to be blocked", actual)
}

func (m *BeDeploymentBlockedMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf("Expected deployment %v not to be blocked", actual)
}

// HaveVulnerability matcher for image scan results
type HaveVulnerabilityMatcher struct {
	expectedSeverity string
	expectedCVE      string
	minCVSS          float64
}

func HaveVulnerability() *HaveVulnerabilityMatcher {
	return &HaveVulnerabilityMatcher{}
}

func (m *HaveVulnerabilityMatcher) WithSeverity(severity string) *HaveVulnerabilityMatcher {
	m.expectedSeverity = severity
	return m
}

func (m *HaveVulnerabilityMatcher) WithCVE(cve string) *HaveVulnerabilityMatcher {
	m.expectedCVE = cve
	return m
}

func (m *HaveVulnerabilityMatcher) WithMinCVSS(score float64) *HaveVulnerabilityMatcher {
	m.minCVSS = score
	return m
}

func (m *HaveVulnerabilityMatcher) Match(actual interface{}) (success bool, err error) {
	scanResult, ok := actual.(*external.ScanResult)
	if !ok {
		return false, fmt.Errorf("HaveVulnerability matcher expects *external.ScanResult")
	}

	for _, vuln := range scanResult.Vulnerabilities {
		// Check severity
		if m.expectedSeverity != "" && vuln.Severity != m.expectedSeverity {
			continue
		}

		// Check CVE
		if m.expectedCVE != "" && vuln.CVE != m.expectedCVE {
			continue
		}

		// Check CVSS score
		if m.minCVSS > 0 && vuln.CVSS < m.minCVSS {
			continue
		}

		// If we get here, vulnerability matches all criteria
		return true, nil
	}

	return false, nil
}

func (m *HaveVulnerabilityMatcher) FailureMessage(actual interface{}) (message string) {
	criteria := []string{}
	if m.expectedSeverity != "" {
		criteria = append(criteria, fmt.Sprintf("severity=%s", m.expectedSeverity))
	}
	if m.expectedCVE != "" {
		criteria = append(criteria, fmt.Sprintf("CVE=%s", m.expectedCVE))
	}
	if m.minCVSS > 0 {
		criteria = append(criteria, fmt.Sprintf("CVSS>=%.1f", m.minCVSS))
	}

	if len(criteria) > 0 {
		return fmt.Sprintf("Expected scan result to have vulnerability with %v", criteria)
	}
	return "Expected scan result to have vulnerability"
}

func (m *HaveVulnerabilityMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf("Expected scan result not to have matching vulnerability")
}

// BeSuccessfulNotification matcher for notification delivery
type BeSuccessfulNotificationMatcher struct{}

func BeSuccessfulNotification() *BeSuccessfulNotificationMatcher {
	return &BeSuccessfulNotificationMatcher{}
}

func (m *BeSuccessfulNotificationMatcher) Match(actual interface{}) (success bool, err error) {
	err, ok := actual.(error)
	if !ok {
		return false, fmt.Errorf("BeSuccessfulNotification matcher expects error from SendMessage")
	}

	// Success is indicated by no error
	return err == nil, nil
}

func (m *BeSuccessfulNotificationMatcher) FailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf("Expected notification to be successful, but got error: %v", actual)
}

func (m *BeSuccessfulNotificationMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	return "Expected notification to fail, but it succeeded"
}

// Eventually helpers for common operations

// EventuallyHaveAlert is a convenience function for Eventually with alert checking
func EventuallyHaveAlert(alertFunc func() []*storage.Alert, count int, timeout time.Duration) types.AsyncAssertion {
	return EventuallyWithOffset(1, alertFunc, timeout, 5*time.Second).Should(HaveAlerts(count))
}

// EventuallyDeploymentBlocked is a convenience function for deployment blocking
func EventuallyDeploymentBlocked(deploymentNameFunc func() string, timeout time.Duration) types.AsyncAssertion {
	return EventuallyWithOffset(1, deploymentNameFunc, timeout, 10*time.Second).Should(BeDeploymentBlocked())
}

// ConsistentlyNoAlert ensures no alerts are generated over a period
func ConsistentlyNoAlert(alertFunc func() []*storage.Alert, duration time.Duration) types.AsyncAssertion {
	return ConsistentlyWithOffset(1, alertFunc, duration, 5*time.Second).Should(HaveAlerts(0))
}

// Helper functions for BDD patterns

// GivenPolicyWithEnforcement creates a BDD Given step for policy creation
func GivenPolicyWithEnforcement(policyName string, enforcement bool) string {
	return fmt.Sprintf("Given a policy '%s' with enforcement=%v", policyName, enforcement)
}

// WhenDeploymentViolatesPolicy creates a BDD When step for deployment creation
func WhenDeploymentViolatesPolicy(deploymentName, policyCategory string) string {
	return fmt.Sprintf("When deployment '%s' violates '%s' policy", deploymentName, policyCategory)
}

// ThenExpectBehavior creates a BDD Then step for expected behavior
func ThenExpectBehavior(shouldBlock, shouldAlert bool, severity string) string {
	behavior := []string{}
	if shouldBlock {
		behavior = append(behavior, "block deployment")
	}
	if shouldAlert {
		behavior = append(behavior, fmt.Sprintf("generate %s alert", severity))
	}
	if len(behavior) == 0 {
		behavior = append(behavior, "allow deployment without alerts")
	}

	return fmt.Sprintf("Then should %s", joinWithAnd(behavior))
}

// Helper function to join strings with "and"
func joinWithAnd(items []string) string {
	if len(items) == 0 {
		return ""
	}
	if len(items) == 1 {
		return items[0]
	}
	if len(items) == 2 {
		return items[0] + " and " + items[1]
	}

	// For more than 2 items, use commas and "and"
	result := ""
	for i, item := range items {
		if i == len(items)-1 {
			result += "and " + item
		} else if i == 0 {
			result += item
		} else {
			result += ", " + item
		}
	}
	return result
}