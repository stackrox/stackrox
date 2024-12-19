package generator

const (
	defaultEmailBodyTemplate = "{{.BrandedPrefix}} has scanned your clusters for compliance with the profiles in your scan configuration." +
		"The attached report lists the checks performed and provides corresponding details to help with remediation. \n" +
		"Profiles:{{.Profile}} |\n" +
		"Passing:{{.Pass}} checks |\n" +
		"Failing:{{.Fail}} checks |\n" +
		"Mixed:{{.Mixed}} checks |\n" +
		"Clusters: {{.Cluster}} scanned"

	defaultSubjectTemplate = "{{.BrandedPrefix}} Compliance Report For {{.ScanConfig}} with {{.Profiles}} Profiles"
)
