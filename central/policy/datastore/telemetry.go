package datastore

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/telemetry/phonehome"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// formatEnumName converts an enum name like "FAIL_BUILD_ENFORCEMENT" to "Fail Build Enforcement"
func formatEnumName(enumName string) string {
	// Remove common suffixes
	enumName = strings.TrimSuffix(enumName, "_ENFORCEMENT")
	enumName = strings.TrimSuffix(enumName, "_SEVERITY")
	enumName = strings.TrimSuffix(enumName, "_EVENT")

	// Replace underscores with spaces and convert to title case
	words := strings.Split(enumName, "_")
	titleCase := cases.Title(language.English, cases.NoLower)
	for i, word := range words {
		words[i] = titleCase.String(strings.ToLower(word))
	}
	return strings.Join(words, " ")
}

// Gather policy configuration and usage metrics.
// Current properties we gather:
// - Total Policies
// - Total Enabled/Disabled Policies
// - Total Custom/Default Policies
// - Total Declarative/Imperative Policies
// - Lifecycle stage distribution
// - Event source distribution
// - Severity distribution
// - Enforcement action usage
// - Policies with scope, exclusions, notifiers
// - Policies with MITRE ATT&CK vectors
// - Average policy complexity metrics
// - Policy criteria field usage (which fields are commonly checked)
var Gather phonehome.GatherFunc = func(ctx context.Context) (map[string]any, error) {
	ctx = sac.WithGlobalAccessScopeChecker(ctx,
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.WorkflowAdministration)))
	props := make(map[string]any)

	policies, err := Singleton().GetAllPolicies(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get policies")
	}

	// Overall counts
	totalPolicies := len(policies)
	enabledCount := 0
	disabledCount := 0
	customCount := 0
	defaultCount := 0
	declarativeCount := 0
	imperativeCount := 0

	// Lifecycle stage counts
	lifecycleDeployCount := 0
	lifecycleBuildCount := 0
	lifecycleRuntimeCount := 0

	// Event source counts - initialize from enum
	eventSourceCounts := make(map[storage.EventSource]int)
	for eventSource := range storage.EventSource_name {
		eventSourceCounts[storage.EventSource(eventSource)] = 0
	}

	// Severity counts - initialize from enum
	severityCounts := make(map[storage.Severity]int)
	for severity := range storage.Severity_name {
		severityCounts[storage.Severity(severity)] = 0
	}

	// Enforcement action counts - initialize from enum
	enforcementActionCounts := make(map[storage.EnforcementAction]int)
	for action := range storage.EnforcementAction_name {
		enforcementActionCounts[storage.EnforcementAction(action)] = 0
	}

	// Feature usage counts
	policiesWithEnforcement := 0
	policiesWithScope := 0
	policiesWithExclusions := 0
	policiesWithNotifiers := 0
	policiesWithCategories := 0
	policiesWithMitreVectors := 0
	policiesWithCriteriaLocked := 0
	policiesWithMitreVectorsLocked := 0

	// Complexity metrics
	totalNotifierCount := 0
	totalCategoryCount := 0
	totalPolicySectionCount := 0
	maxPolicySections := 0

	// Policy criteria field usage (which fields are checked in policy groups)
	policyFieldCounts := make(map[string]int)

	for _, policy := range policies {
		// Overall counts
		if policy.GetDisabled() {
			disabledCount++
		} else {
			enabledCount++
		}

		if policy.GetIsDefault() {
			defaultCount++
		} else {
			customCount++
		}

		if policy.GetSource() == storage.PolicySource_DECLARATIVE {
			declarativeCount++
		} else {
			imperativeCount++
		}

		// Lifecycle stages
		for _, stage := range policy.GetLifecycleStages() {
			switch stage {
			case storage.LifecycleStage_DEPLOY:
				lifecycleDeployCount++
			case storage.LifecycleStage_BUILD:
				lifecycleBuildCount++
			case storage.LifecycleStage_RUNTIME:
				lifecycleRuntimeCount++
			}
		}

		// Event source
		eventSourceCounts[policy.GetEventSource()]++

		// Severity
		severityCounts[policy.GetSeverity()]++

		// Enforcement actions
		hasEnforcement := false
		for _, action := range policy.GetEnforcementActions() {
			if action != storage.EnforcementAction_UNSET_ENFORCEMENT {
				enforcementActionCounts[action]++
				hasEnforcement = true
			}
		}
		if hasEnforcement {
			policiesWithEnforcement++
		}

		// Feature usage
		if len(policy.GetScope()) > 0 {
			policiesWithScope++
		}

		if len(policy.GetExclusions()) > 0 {
			policiesWithExclusions++
		}

		if len(policy.GetNotifiers()) > 0 {
			policiesWithNotifiers++
			totalNotifierCount += len(policy.GetNotifiers())
		}

		if len(policy.GetCategories()) > 0 {
			policiesWithCategories++
			totalCategoryCount += len(policy.GetCategories())
		}

		if len(policy.GetMitreAttackVectors()) > 0 {
			policiesWithMitreVectors++
		}

		if policy.GetCriteriaLocked() {
			policiesWithCriteriaLocked++
		}

		if policy.GetMitreVectorsLocked() {
			policiesWithMitreVectorsLocked++
		}

		// Policy sections complexity
		sectionCount := len(policy.GetPolicySections())
		totalPolicySectionCount += sectionCount
		if sectionCount > maxPolicySections {
			maxPolicySections = sectionCount
		}

		// Count policy field names used in policy groups
		for _, section := range policy.GetPolicySections() {
			for _, group := range section.GetPolicyGroups() {
				if fieldName := group.GetFieldName(); fieldName != "" {
					policyFieldCounts[fieldName]++
				}
			}
		}
	}

	// Add total policies
	_ = phonehome.AddTotal(ctx, props, "Policies", phonehome.Constant(totalPolicies))

	// Overall counts
	props["Total Enabled Policies"] = enabledCount
	props["Total Disabled Policies"] = disabledCount
	props["Total Custom Policies"] = customCount
	props["Total Default Policies"] = defaultCount
	props["Total Declarative Policies"] = declarativeCount
	props["Total Imperative Policies"] = imperativeCount

	// Lifecycle stage distribution
	props["Policies with Deploy Lifecycle"] = lifecycleDeployCount
	props["Policies with Build Lifecycle"] = lifecycleBuildCount
	props["Policies with Runtime Lifecycle"] = lifecycleRuntimeCount

	// Event source distribution
	for eventSource, count := range eventSourceCounts {
		eventSourceName := storage.EventSource_name[int32(eventSource)]
		props[fmt.Sprintf("Policies with %s Event", formatEnumName(eventSourceName))] = count
	}

	// Severity distribution
	for severity, count := range severityCounts {
		severityName := storage.Severity_name[int32(severity)]
		props[fmt.Sprintf("%s Policies", formatEnumName(severityName))] = count
	}

	// Enforcement usage
	props["Policies with Enforcement"] = policiesWithEnforcement
	for action, count := range enforcementActionCounts {
		// Skip UNSET_ENFORCEMENT as it's not a real enforcement action
		if action == storage.EnforcementAction_UNSET_ENFORCEMENT {
			continue
		}
		actionName := storage.EnforcementAction_name[int32(action)]
		props[fmt.Sprintf("Policies with %s", formatEnumName(actionName))] = count
	}

	// Feature usage
	props["Policies with Scope Defined"] = policiesWithScope
	props["Policies with Exclusions"] = policiesWithExclusions
	props["Policies with Notifiers"] = policiesWithNotifiers
	props["Policies with Categories"] = policiesWithCategories
	props["Policies with MITRE ATT&CK Vectors"] = policiesWithMitreVectors
	props["Policies with Criteria Locked"] = policiesWithCriteriaLocked
	props["Policies with MITRE Vectors Locked"] = policiesWithMitreVectorsLocked

	// Average complexity metrics
	if totalPolicies > 0 {
		props["Avg Notifiers per Policy"] = fmt.Sprintf("%.2f", float64(totalNotifierCount)/float64(totalPolicies))
		props["Avg Categories per Policy"] = fmt.Sprintf("%.2f", float64(totalCategoryCount)/float64(totalPolicies))
		props["Avg Policy Sections per Policy"] = fmt.Sprintf("%.2f", float64(totalPolicySectionCount)/float64(totalPolicies))
	} else {
		props["Avg Notifiers per Policy"] = "0.00"
		props["Avg Categories per Policy"] = "0.00"
		props["Avg Policy Sections per Policy"] = "0.00"
	}
	props["Max Policy Sections in a Policy"] = maxPolicySections

	// Policy criteria field usage - report which fields are most commonly used
	titleCase := cases.Title(language.English, cases.NoLower)
	for fieldName, count := range policyFieldCounts {
		// Format field name for readability (e.g., "CVE" -> "Cve", "Image Age" -> "Image Age")
		formattedName := titleCase.String(strings.ToLower(strings.ReplaceAll(fieldName, "_", " ")))
		props[fmt.Sprintf("Policy Criteria %s Usage", formattedName)] = count
	}

	return props, nil
}
