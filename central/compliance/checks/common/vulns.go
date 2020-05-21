package common

import (
	"strings"
	"unicode"

	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/pkg/booleanpolicy/policyfields"
	"github.com/stackrox/rox/pkg/set"
)

// To match more than one CVE, the regex must contain
// SOME character that's not a letter, or a -, or a number.
func checkCVEIsNotSingleRegexMatch(cveValue string) bool {
	if cveValue == "" {
		return false
	}
	return strings.IndexFunc(cveValue, func(r rune) bool {
		return r != '-' && !unicode.IsLetter(r) && !unicode.IsNumber(r)
	}) >= 0
}

// CheckAtLeastOnePolicyEnabledReferringToVulnsInterpretation is reusable interpretation text for CheckAtLeastOnePolicyEnabledReferringToVulns.
const CheckAtLeastOnePolicyEnabledReferringToVulnsInterpretation = `StackRox checks that at least one policy is enabled for image vulnerabilities (using a CVE ID pattern or CVSS score comparison).`

// CheckAtLeastOnePolicyEnabledReferringToVulns does the following:
// Verify at least one policy is enabled referring to vulnerabilities.
// Don’t give credit for specific CVE matches if possible (e.g., built-in Struts policy
// should not match, but CVSS-based or CVE .* policies should).
// As a convenience, it also returns the ids of the policies that refer to vulns.
func CheckAtLeastOnePolicyEnabledReferringToVulns(ctx framework.ComplianceContext) set.StringSet {
	vulnPolicyIDs := set.NewStringSet()
	for _, policy := range ctx.Data().Policies() {
		if !IsPolicyEnabled(policy) {
			continue
		}
		if policyfields.ContainsCVSSField(policy) {
			vulnPolicyIDs.Add(policy.GetId())
			framework.Passf(ctx, "Policy %q is enabled, and targets vulnerabilities", policy.GetName())
			continue
		}
		for _, cveField := range policyfields.GetCVEs(policy) {
			if checkCVEIsNotSingleRegexMatch(cveField) {
				vulnPolicyIDs.Add(policy.GetId())
				framework.Passf(ctx, "Policy %q is enabled, and targets vulnerabilities", policy.GetName())
				break
			}
		}
	}
	if vulnPolicyIDs.Cardinality() == 0 {
		framework.Fail(ctx, "No policies referring to vulnerabilities (using a CVE ID pattern or CVSS score comparison) were enabled")
	}
	return vulnPolicyIDs
}
