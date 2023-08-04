package reportgenerator

const (
	paginationLimit = 50

	cveFieldsFragment = `fragment cveFields on ImageVulnerability {
                             cve
	                         severity
                             fixedByVersion
                             isFixable
                             cvss
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

	vulnReportEmailTemplate = `
	{{.BrandedProductName}} has found vulnerabilities associated with the {{.ImageTypes}} images owned by your organization. Please review the attached vulnerability report {{.WhichVulns}} for {{.DateStr}}.

	To address these findings, please review the impacted software packages in the images you are responsible for and update them to a version containing the fix, if one is available.`

	noVulnsFoundEmailTemplate = `
	{{.BrandedProductName}} has found zero vulnerabilities associated with the {{.ImageTypes}} images owned by your organization.`

	paginatedQueryStartOffset = 0
)
