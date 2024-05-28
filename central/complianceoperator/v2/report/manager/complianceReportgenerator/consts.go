package complianceReportgenerator

const (
<<<<<<< HEAD
	defaultEmailBodyTemplate = "{{.BrandedPrefix}} has scanned your clusters for compliance with the profiles in your scan configuration." +
		"The attached report lists the checks performed and provides corresponding details to help with remediation. \n" +
=======
	defaultEmailBodyTemplate = "{{.BrandedPrefix}} has identified {{.ComplianceStatus}} profile checks for clusters scanned by your \n" +
		"schedule configuration parameters. The attached report lists those checks and associated details to aid with remediation. \n" +
>>>>>>> 6faeddcd64 (Added test file)
		"Profiles:{{.Profile}} \n" +
		"Passing:{{.Pass}} checks \n" +
		"Failing:{{.Fail}} checks \n" +
		"Mixed:{{.Mixed}} checks \n" +
		"Clusters {{.Clusters}} scanned"

<<<<<<< HEAD
	defaultSubjectTemplate = "{{.BrandedPrefix}} Compliance Report For {{.ScanConfig}} with {{.Profiles}} Profiles"
=======
	defaultSubjectTemplate = "{{.BrandedPrefix}} Compliance Report For {{.ScanConfig}} Profiles {{.Profiles}}"
>>>>>>> 6faeddcd64 (Added test file)
)
