package reportgenerator

const (
	defaultEmailBodyTemplate = "{{.BrandedPrefix}} has identified non compliant profile checks for clusters scanned by your \n" +
		"schedule configuration parameters. The attached report lists those checks and associated details to aid with remediation. \n" +
		"Profiles:{{.Profile}} \n" +
		"Passing:{{.Pass}} checks \n" +
		"Failing:{{.Fail}} checks \n" +
		"Mixed:{{.Mixed}} checks \n" +
		"Clusters {{.Clusters}} scanned"

	defaultSubjectTemplate = "{{.BrandedPrefix}} Compliance Report For {{.ScanConfig}} Profiles {{.Profiles}}"
)
