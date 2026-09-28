package reportgenerator

const (
	defaultEmailSubjectTemplate = "{{.BrandedProductNameShort}} Workload CVE Report for {{.ReportConfigName}}; Scope: {{.CollectionName}}"

	defaultEmailBodyTemplate = "{{.BrandedPrefix}} for Kubernetes has identified workload CVEs in the images matched by the following report configuration parameters. " +
		"The attached Vulnerability report lists those workload CVEs and associated details to help with remediation. " +
		"Please review the vulnerable software packages/components from the impacted images and update them to a version containing the fix, if one is available.\n"

	defaultNoVulnsEmailBodyTemplate = "{{.BrandedPrefix}} for Kubernetes has found no workload CVEs in the images matched by the following report configuration parameters.\n"

	// NodeDefaultEmailSubjectTemplate is the default email subject for node CVE reports.
	NodeDefaultEmailSubjectTemplate = "{{.BrandedProductNameShort}} Node CVE Report for {{.ReportConfigName}}"

	// NodeDefaultEmailBodyTemplate is the default email body for node CVE reports.
	NodeDefaultEmailBodyTemplate = "{{.BrandedPrefix}} for Kubernetes has identified node CVEs in the nodes matched by the following report configuration parameters. " +
		"The attached vulnerability report lists those node CVEs and associated details to help with remediation. " +
		"Please review the vulnerable software packages/components from the impacted nodes and update them to a version containing the fix, if one is available.\n"

	// NodeDefaultNoVulnsEmailBodyTemplate is the default email body when no node CVEs are found.
	NodeDefaultNoVulnsEmailBodyTemplate = "{{.BrandedPrefix}} for Kubernetes has found no node CVEs in the nodes matched by the following report configuration parameters.\n"
)
