// Package reportcheck performs lightweight diagnostics on VMReports
// passing through Sensor. It logs warnings for common data-quality
// issues (missing repos, empty CPEs, orphaned packages) so operators
// can fix guest-side configuration without digging into Central.
//
// Usage: reportcheck.Log(report) — one call, zero coupling.
package reportcheck

import (
	v1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/pkg/logging"
)

var log = logging.LoggerForModule()

// Log inspects the report and emits warnings for anything that would
// cause Central/Scanner to mark components as UNSCANNED.
func Log(vmLabel string, report *v1.VMReport) {
	ir := report.GetIndexReport().GetIndexV4()
	if ir == nil {
		log.Warnf("VM report check [%s]: index_v4 is nil — nothing to analyze", vmLabel)
		return
	}
	if !ir.GetSuccess() {
		log.Warnf("VM report check [%s]: indexer reported failure: %s", vmLabel, ir.GetErr())
		return
	}

	c := ir.GetContents()
	if c == nil {
		log.Warnf("VM report check [%s]: contents is nil", vmLabel)
		return
	}

	pkgs := c.GetPackages()
	repos := c.GetRepositories()
	envs := c.GetEnvironments()

	if len(repos) == 0 {
		log.Warnf("VM report check [%s]: 0 repositories — roxagent could not discover RPM repo metadata. "+
			"Check that /etc/yum.repos.d is bind-mounted into the scan root and repos are enabled (subscription-manager).",
			vmLabel)
	}

	var reposWithCPE int
	for _, r := range repos {
		if r.GetCpe() != "" {
			reposWithCPE++
		}
	}
	if len(repos) > 0 && reposWithCPE == 0 {
		log.Warnf("VM report check [%s]: %d repositories found but none have a CPE — "+
			"repo-to-CPE mapping failed. Check that --repo-cpe-url is reachable from inside the VM.",
			vmLabel, len(repos))
	}

	if len(envs) == 0 && len(pkgs) > 0 {
		log.Warnf("VM report check [%s]: %d packages but 0 environments — "+
			"packages are not linked to any repository; all will be UNSCANNED.",
			vmLabel, len(pkgs))
	}

	var linked, orphaned int
	for pkgID := range pkgs {
		if el := envs[pkgID]; el != nil && len(el.GetEnvironments()) > 0 {
			linked++
		} else {
			orphaned++
		}
	}

	if orphaned > 0 {
		log.Warnf("VM report check [%s]: %d/%d packages have no environment entry — they will be UNSCANNED.",
			vmLabel, orphaned, len(pkgs))
	}

	log.Infof("VM report check [%s]: %d packages, %d repositories (%d with CPE), %d linked, %d orphaned",
		vmLabel, len(pkgs), len(repos), reposWithCPE, linked, orphaned)
}
