package checksi22

import (
	"strings"
	"unicode"

	"github.com/stackrox/rox/central/compliance/checks/common"
	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/pkg/features"
)

const (
	controlID = `NIST_SP_800_53:SI_2_(2)`

	interpretationText = `This control requires that system flaws be identified and remediated in a timely manner.

For this control, ` + common.AllDeployedImagesHaveMatchingIntegrationsInterpretation + `

StackRox also checks that at least one policy is enabled for image vulnerabilities (using a CVE ID pattern or CVSS score comparison).`
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

// Verify at least one policy is enabled referring to vulnerabilities.
// Don’t give credit for specific CVE matches if possible (e.g., built-in Struts policy
// should not match, but CVSS-based or CVE .* policies should).
func checkAtLeastOnePolicyEnabledReferringToVulns(ctx framework.ComplianceContext) {
	var passed bool
	for _, policy := range ctx.Data().Policies() {
		if !common.IsPolicyEnabled(policy) {
			continue
		}
		if policy.GetFields().GetCvss() != nil {
			passed = true
			framework.Passf(ctx, "Policy %q is enabled, and targets vulnerabilities", policy.GetName())
		}
		if checkCVEIsNotSingleRegexMatch(policy.GetFields().GetCve()) {
			passed = true
			framework.Passf(ctx, "Policy %q is enabled, and targets vulnerabilities", policy.GetName())
		}
	}
	if !passed {
		framework.Fail(ctx, "No policies referring to vulnerabilities (using a CVE ID pattern or CVSS score comparison) were enabled")
	}
}

func init() {
	framework.MustRegisterNewCheckIfFlagEnabled(
		framework.CheckMetadata{
			ID:                 controlID,
			Scope:              framework.ClusterKind,
			DataDependencies:   []string{"Policies"},
			InterpretationText: interpretationText,
		},
		func(ctx framework.ComplianceContext) {
			common.CheckAllDeployedImagesHaveMatchingIntegrations(ctx)
			checkAtLeastOnePolicyEnabledReferringToVulns(ctx)
		}, features.NistSP800_53)
}
