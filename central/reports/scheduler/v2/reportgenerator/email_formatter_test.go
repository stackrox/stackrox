package reportgenerator

import (
	"strings"
	"testing"

	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/timeutil"
	"github.com/stretchr/testify/suite"
)

type configDetailsTestCase struct {
	desc         string
	snapshot     *storage.ReportSnapshot
	expectedHtml string
}

func TestEmailFormatter(t *testing.T) {
	suite.Run(t, new(EmailFormatterTestSuite))
}

type EmailFormatterTestSuite struct {
	suite.Suite
}

func (s *EmailFormatterTestSuite) SetupSuite() {
	s.T().Setenv(env.VulnReportingEnhancements.EnvVar(), "true")
	if !env.VulnReportingEnhancements.BooleanSetting() {
		s.T().Skip("Skip tests when ROX_VULN_MGMT_REPORTING_ENHANCEMENTS disabled")
		s.T().SkipNow()
	}
}

func (s *EmailFormatterTestSuite) TestFormatReportConfigDetails() {
	for _, tc := range s.configDetailsTestCases() {
		s.T().Run(tc.desc, func(t *testing.T) {
			configHtml, err := formatReportConfigDetails(tc.snapshot)
			s.Require().NoError(err)
			expectedHtml := strings.ReplaceAll(tc.expectedHtml, "\n", "")
			expectedHtml = strings.ReplaceAll(expectedHtml, "\t", "")
			s.Require().Equal(expectedHtml, configHtml)
		})
	}
}

func (s *EmailFormatterTestSuite) configDetailsTestCases() []configDetailsTestCase {
	cases := []configDetailsTestCase{
		{
			desc:     "All severities, image types, fixabilities; Cves since last scheduled report",
			snapshot: testReportSnapshot(),
			expectedHtml: `<html>
				<body>
					<table style="width: 100%; border-collapse: collapse; table-layout: fixed; border: none; text-align: left;">
						<tr>
							<th style="background-color: #f0f0f0; padding: 10px;">CVE Severity</th>
							<th style="background-color: #f0f0f0; padding: 10px;">CVE Status</th>
						</tr>
						<tr>
							<td style="padding: 10px; word-wrap: break-word; white-space: normal;">
								<table style="width: 100%; border-collapse: collapse; table-layout: fixed; border: none; text-align: left;">
									<tr><td style="padding: 10px;">Critical</td></tr>
									<tr><td style="padding: 10px;">Important</td></tr>
									<tr><td style="padding: 10px;">Moderate</td></tr>
									<tr><td style="padding: 10px;">Low</td></tr>
								</table>
							</td>
				
							<td style="padding: 10px; word-wrap: break-word; white-space: normal;">
								<table style="width: 100%; border-collapse: collapse; table-layout: fixed; border: none; text-align: left;">
									<tr><td style="padding: 10px;">Fixable</td></tr>
									<tr><td style="padding: 10px;">Not Fixable</td></tr>
								</table>
							</td>
						</tr>
						<tr>
							<th style="background-color: #f0f0f0; padding: 10px;">Report Scope</th>
							<th style="background-color: #f0f0f0; padding: 10px;">Image Type</th>
							<th style="background-color: #f0f0f0; padding: 10px;">CVEs discovered since</th>
						</tr>
						<tr>
							<td style="padding: 10px; word-wrap: break-word; white-space: normal;">
								<table style="width: 100%; border-collapse: collapse; table-layout: fixed; border: none; text-align: left;">
									<tr>
										<td style="padding: 10px;">
											collection-1
										</td>
									</tr>
								</table>
							</td>
							<td style="padding: 10px; word-wrap: break-word; white-space: normal;">
								<table style="width: 100%; border-collapse: collapse; table-layout: fixed; border: none; text-align: left;">
									<tr><td style="padding: 10px;">Deployed Images</td></tr>
									<tr><td style="padding: 10px;">Watched Images</td></tr>
								</table>
							</td>
							<td style="padding: 10px; word-wrap: break-word; white-space: normal;">
								<table style="width: 100%; border-collapse: collapse; table-layout: fixed; border: none; text-align: left;">
									<tr><td style="padding: 10px;">Last successful scheduled report</td></tr>
								</table>
							</td>
						</tr>
					</table>
				</body>
				</html>`,
		},
		{
			desc: "All severities, image types, fixabilities; Cves since All time",
			snapshot: func() *storage.ReportSnapshot {
				snap := testReportSnapshot()
				snap.GetVulnReportFilters().CvesSince = &storage.VulnerabilityReportFilters_AllVuln{
					AllVuln: true,
				}
				return snap
			}(),
			expectedHtml: `<html>
				<body>
					<table style="width: 100%; border-collapse: collapse; table-layout: fixed; border: none; text-align: left;">
						<tr>
							<th style="background-color: #f0f0f0; padding: 10px;">CVE Severity</th>
							<th style="background-color: #f0f0f0; padding: 10px;">CVE Status</th>
						</tr>
						<tr>
							<td style="padding: 10px; word-wrap: break-word; white-space: normal;">
								<table style="width: 100%; border-collapse: collapse; table-layout: fixed; border: none; text-align: left;">
									<tr><td style="padding: 10px;">Critical</td></tr>
									<tr><td style="padding: 10px;">Important</td></tr>
									<tr><td style="padding: 10px;">Moderate</td></tr>
									<tr><td style="padding: 10px;">Low</td></tr>
								</table>
							</td>
				
							<td style="padding: 10px; word-wrap: break-word; white-space: normal;">
								<table style="width: 100%; border-collapse: collapse; table-layout: fixed; border: none; text-align: left;">
									<tr><td style="padding: 10px;">Fixable</td></tr>
									<tr><td style="padding: 10px;">Not Fixable</td></tr>
								</table>
							</td>
						</tr>
						<tr>
							<th style="background-color: #f0f0f0; padding: 10px;">Report Scope</th>
							<th style="background-color: #f0f0f0; padding: 10px;">Image Type</th>
							<th style="background-color: #f0f0f0; padding: 10px;">CVEs discovered since</th>
						</tr>
						<tr>
							<td style="padding: 10px; word-wrap: break-word; white-space: normal;">
								<table style="width: 100%; border-collapse: collapse; table-layout: fixed; border: none; text-align: left;">
									<tr>
										<td style="padding: 10px;">
											collection-1
										</td>
									</tr>
								</table>
							</td>
							<td style="padding: 10px; word-wrap: break-word; white-space: normal;">
								<table style="width: 100%; border-collapse: collapse; table-layout: fixed; border: none; text-align: left;">
									<tr><td style="padding: 10px;">Deployed Images</td></tr>
									<tr><td style="padding: 10px;">Watched Images</td></tr>
								</table>
							</td>
							<td style="padding: 10px; word-wrap: break-word; white-space: normal;">
								<table style="width: 100%; border-collapse: collapse; table-layout: fixed; border: none; text-align: left;">
									<tr><td style="padding: 10px;">All Time</td></tr>
								</table>
							</td>
						</tr>
					</table>
				</body>
				</html>`,
		},
		{
			desc: "Critical severity, fixable CVEs, Watched Images; Cves since custom date",
			snapshot: func() *storage.ReportSnapshot {
				snap := testReportSnapshot()
				snap.GetVulnReportFilters().Severities = []storage.VulnerabilitySeverity{
					storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
				}
				snap.GetVulnReportFilters().Fixability = storage.VulnerabilityReportFilters_FIXABLE
				snap.GetVulnReportFilters().ImageTypes = []storage.VulnerabilityReportFilters_ImageType{
					storage.VulnerabilityReportFilters_DEPLOYED,
				}
				dateTs, err := types.TimestampProto(timeutil.MustParse("2006-01-02 15:04:05", "2023-01-20 22:42:02"))
				s.Require().NoError(err)
				snap.GetVulnReportFilters().CvesSince = &storage.VulnerabilityReportFilters_SinceStartDate{
					SinceStartDate: dateTs,
				}
				return snap
			}(),
			expectedHtml: `<html>
				<body>
					<table style="width: 100%; border-collapse: collapse; table-layout: fixed; border: none; text-align: left;">
						<tr>
							<th style="background-color: #f0f0f0; padding: 10px;">CVE Severity</th>
							<th style="background-color: #f0f0f0; padding: 10px;">CVE Status</th>
						</tr>
						<tr>
							<td style="padding: 10px; word-wrap: break-word; white-space: normal;">
								<table style="width: 100%; border-collapse: collapse; table-layout: fixed; border: none; text-align: left;">
									<tr><td style="padding: 10px;">Critical</td></tr>
								</table>
							</td>
				
							<td style="padding: 10px; word-wrap: break-word; white-space: normal;">
								<table style="width: 100%; border-collapse: collapse; table-layout: fixed; border: none; text-align: left;">
									<tr><td style="padding: 10px;">Fixable</td></tr>
								</table>
							</td>
						</tr>
						<tr>
							<th style="background-color: #f0f0f0; padding: 10px;">Report Scope</th>
							<th style="background-color: #f0f0f0; padding: 10px;">Image Type</th>
							<th style="background-color: #f0f0f0; padding: 10px;">CVEs discovered since</th>
						</tr>
						<tr>
							<td style="padding: 10px; word-wrap: break-word; white-space: normal;">
								<table style="width: 100%; border-collapse: collapse; table-layout: fixed; border: none; text-align: left;">
									<tr>
										<td style="padding: 10px;">
											collection-1
										</td>
									</tr>
								</table>
							</td>
							<td style="padding: 10px; word-wrap: break-word; white-space: normal;">
								<table style="width: 100%; border-collapse: collapse; table-layout: fixed; border: none; text-align: left;">
									<tr><td style="padding: 10px;">Deployed Images</td></tr>
								</table>
							</td>
							<td style="padding: 10px; word-wrap: break-word; white-space: normal;">
								<table style="width: 100%; border-collapse: collapse; table-layout: fixed; border: none; text-align: left;">
									<tr><td style="padding: 10px;">January 20, 2023</td></tr>
								</table>
							</td>
						</tr>
					</table>
				</body>
				</html>`,
		},
	}
	return cases
}

func testReportSnapshot() *storage.ReportSnapshot {
	return &storage.ReportSnapshot{
		ReportConfigurationId: "config-1",
		Name:                  "config-1",
		Collection: &storage.CollectionSnapshot{
			Id:   "collection-1",
			Name: "collection-1",
		},
		Filter: &storage.ReportSnapshot_VulnReportFilters{
			VulnReportFilters: &storage.VulnerabilityReportFilters{
				Fixability: storage.VulnerabilityReportFilters_BOTH,
				Severities: []storage.VulnerabilitySeverity{
					storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY,
					storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
					storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY,
					storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
				},
				ImageTypes: []storage.VulnerabilityReportFilters_ImageType{
					storage.VulnerabilityReportFilters_DEPLOYED,
					storage.VulnerabilityReportFilters_WATCHED,
				},
				CvesSince: &storage.VulnerabilityReportFilters_SinceLastSentScheduledReport{
					SinceLastSentScheduledReport: true,
				},
			},
		},
	}
}
