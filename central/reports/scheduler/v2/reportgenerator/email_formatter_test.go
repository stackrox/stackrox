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
			configHtml, err := formatReportConfigDetails(tc.snapshot, 50, 30)
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
			expectedHtml: `<div>
						<div style="padding: 0 0 10px 0">
							<span style="font-weight: bold; margin-right: 10px">
								Config name: 
							</span>
							<span>config-1</span>
						</div>
						<div style="padding: 0 0 10px 0">
							<span style="font-weight: bold; margin-right: 10px">
								Number of CVEs found: 
							</span>
							<span>50 in Deployed images, 30 in Watched images</span>
						</div>
						<div style="padding: 0 0 10px 0">
							<span style="font-weight: bold; margin-right: 10px">
								CVE severity: 
							</span>
							<span>Critical, Important, Moderate, Low</span>
						</div>
						<div style="padding: 0 0 10px 0">
							<span style="font-weight: bold; margin-right: 10px">
								CVE status: 
							</span>
							<span>Fixable, Not fixable</span>
						</div>
						<div style="padding: 0 0 10px 0">
							<span style="font-weight: bold; margin-right: 10px">
								Report scope: 
							</span>
							<span>collection-1</span>
						</div>
						<div style="padding: 0 0 10px 0">
							<span style="font-weight: bold; margin-right: 10px">
								Image type: 
							</span>
							<span>Deployed images, Watched images</span>
						</div>
						<div style="padding: 0 0 10px 0">
							<span style="font-weight: bold; margin-right: 10px">
								CVEs discovered since: 
							</span>
							<span>Last successful scheduled report</span>
						</div>
					</div>`,
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
			expectedHtml: `<div>
						<div style="padding: 0 0 10px 0">
							<span style="font-weight: bold; margin-right: 10px">
								Config name: 
							</span>
							<span>config-1</span>
						</div>
						<div style="padding: 0 0 10px 0">
							<span style="font-weight: bold; margin-right: 10px">
								Number of CVEs found: 
							</span>
							<span>50 in Deployed images, 30 in Watched images</span>
						</div>
						<div style="padding: 0 0 10px 0">
							<span style="font-weight: bold; margin-right: 10px">
								CVE severity: 
							</span>
							<span>Critical, Important, Moderate, Low</span>
						</div>
						<div style="padding: 0 0 10px 0">
							<span style="font-weight: bold; margin-right: 10px">
								CVE status: 
							</span>
							<span>Fixable, Not fixable</span>
						</div>
						<div style="padding: 0 0 10px 0">
							<span style="font-weight: bold; margin-right: 10px">
								Report scope: 
							</span>
							<span>collection-1</span>
						</div>
						<div style="padding: 0 0 10px 0">
							<span style="font-weight: bold; margin-right: 10px">
								Image type: 
							</span>
							<span>Deployed images, Watched images</span>
						</div>
						<div style="padding: 0 0 10px 0">
							<span style="font-weight: bold; margin-right: 10px">
								CVEs discovered since: 
							</span>
							<span>All time</span>
						</div>
					</div>`,
		},
		{
			desc: "Critical severity, fixable CVEs, Deployed Images; Cves since custom date",
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
			expectedHtml: `<div>
						<div style="padding: 0 0 10px 0">
							<span style="font-weight: bold; margin-right: 10px">
								Config name: 
							</span>
							<span>config-1</span>
						</div>
						<div style="padding: 0 0 10px 0">
							<span style="font-weight: bold; margin-right: 10px">
								Number of CVEs found: 
							</span>
							<span>50 in Deployed images, 30 in Watched images</span>
						</div>
						<div style="padding: 0 0 10px 0">
							<span style="font-weight: bold; margin-right: 10px">
								CVE severity: 
							</span>
							<span>Critical</span>
						</div>
						<div style="padding: 0 0 10px 0">
							<span style="font-weight: bold; margin-right: 10px">
								CVE status: 
							</span>
							<span>Fixable</span>
						</div>
						<div style="padding: 0 0 10px 0">
							<span style="font-weight: bold; margin-right: 10px">
								Report scope: 
							</span>
							<span>collection-1</span>
						</div>
						<div style="padding: 0 0 10px 0">
							<span style="font-weight: bold; margin-right: 10px">
								Image type: 
							</span>
							<span>Deployed images</span>
						</div>
						<div style="padding: 0 0 10px 0">
							<span style="font-weight: bold; margin-right: 10px">
								CVEs discovered since: 
							</span>
							<span>January 20, 2023</span>
						</div>
					</div>`,
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
