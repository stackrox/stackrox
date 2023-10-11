package reportgenerator

const (
	paginationLimit = 50

	cveFieldsFragment = `fragment cveFields on ImageVulnerability {
                             cve
	                         severity
                             fixedByVersion
                             isFixable
                             discoveredAtImage
		                     link
                             cvss
                         }`

	deployedImagesReportQuery = `query getDeployedImagesReportData($scopequery: String, 
                               $cvequery: String, $pagination: Pagination) {
							       deployments: deployments(query: $scopequery, pagination: $pagination) {
                                       clusterName
								       namespace
                                       name
                                       images {
                                           name {
								               full_name:fullName
								           }
                                           imageComponents {
									           name
									           imageVulnerabilities(query: $cvequery) {
										           ...cveFields
									           }
								           }
							           }
						           }
					           }` +
		cveFieldsFragment
	deployedImagesReportQueryOpName = "getDeployedImagesReportData"

	watchedImagesReportQuery = `query getWatchedImagesReportData($scopequery: String, $cvequery: String, $pagination: Pagination) {
                              images: images(query: $scopequery, pagination: $pagination) {
                                  name {
                                      full_name:fullName
                                  }
                                  imageComponents {
                                      name
                                      imageVulnerabilities(query: $cvequery) {
                                          ...cveFields
                                      }
                                  }
                              }
                          }` +
		cveFieldsFragment
	watchedImagesReportQueryOpName = "getWatchedImagesReportData"

	defaultEmailSubjectTemplate = "{{.BrandedProductNameShort}} Workload CVE Report for {{.ReportConfigName}}; Scope: {{.CollectionName}}"

	defaultEmailBodyTemplate = "{{.BrandedPrefix}} for Kubernetes has identified workload CVEs in the images matched by the following report configuration parameters. " +
		"The attached Vulnerability report lists those workload CVEs and associated details to help with remediation. " +
		"Please review the vulnerable software packages/components from the impacted images and update them to a version containing the fix, if one is available.\n"

	defaultNoVulnsEmailBodyTemplate = "{{.BrandedPrefix}} for Kubernetes has found no workload CVEs in the images matched by the following report configuration parameters.\n"

	paginatedQueryStartOffset = 0
)
