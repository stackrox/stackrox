package scannerv4

import v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"

// normalizeNodeIndexCPEs copies the report and expands only the exact shortened
// wildcard sent by older Sensors. Scanner's CPE 2.3 parser requires all eleven
// attributes. Nil reports, nil elements, and all other CPE strings are preserved.
func normalizeNodeIndexCPEs(report *v4.IndexReport) *v4.IndexReport {
	report = report.CloneVT()
	contents := report.GetContents()
	for _, pkg := range contents.GetPackages() {
		normalizeNodePackageCPEs(pkg)
	}
	for _, pkg := range contents.GetPackagesDEPRECATED() {
		normalizeNodePackageCPEs(pkg)
	}
	for _, dist := range contents.GetDistributions() {
		if dist != nil {
			dist.Cpe = normalizeLegacyNodeCPE(dist.GetCpe())
		}
	}
	for _, dist := range contents.GetDistributionsDEPRECATED() {
		if dist != nil {
			dist.Cpe = normalizeLegacyNodeCPE(dist.GetCpe())
		}
	}
	for _, repo := range contents.GetRepositories() {
		if repo != nil {
			repo.Cpe = normalizeLegacyNodeCPE(repo.GetCpe())
		}
	}
	for _, repo := range contents.GetRepositoriesDEPRECATED() {
		if repo != nil {
			repo.Cpe = normalizeLegacyNodeCPE(repo.GetCpe())
		}
	}
	return report
}

func normalizeNodePackageCPEs(pkg *v4.Package) {
	for ; pkg != nil; pkg = pkg.GetSource() {
		pkg.Cpe = normalizeLegacyNodeCPE(pkg.GetCpe())
	}
}

func normalizeLegacyNodeCPE(s string) string {
	if s == "cpe:2.3:*" {
		return unsetCPE
	}
	return s
}
