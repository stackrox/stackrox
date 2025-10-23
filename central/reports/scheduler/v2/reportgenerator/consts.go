package reportgenerator

const (
	defaultEmailSubjectTemplate = "{{.BrandedProductNameShort}} Workload CVE Report for {{.ReportConfigName}}; Scope: {{.CollectionName}}"

	defaultEmailBodyTemplate = "{{.BrandedPrefix}} for Kubernetes has identified workload CVEs in the images matched by the following report configuration parameters. " +
		"The attached Vulnerability report lists those workload CVEs and associated details to help with remediation. " +
		"Please review the vulnerable software packages/components from the impacted images and update them to a version containing the fix, if one is available.\n"

	defaultNoVulnsEmailBodyTemplate = "{{.BrandedPrefix}} for Kubernetes has found no workload CVEs in the images matched by the following report configuration parameters.\n"
)
