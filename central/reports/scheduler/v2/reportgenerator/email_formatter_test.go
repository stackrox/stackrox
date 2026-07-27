package reportgenerator

import (
	"strings"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/timeutil"
	"github.com/stretchr/testify/suite"
)

type configDetailsTestCase struct {
	desc         string
	snapshot     *storage.ReportSnapshot
	expectedHTML string
}

func TestEmailFormatter(t *testing.T) {
	suite.Run(t, new(EmailFormatterTestSuite))
}

type EmailFormatterTestSuite struct {
	suite.Suite
}

func (s *EmailFormatterTestSuite) TestFormatReportConfigDetails() {
	for _, tc := range s.configDetailsTestCases() {
		s.T().Run(tc.desc, func(t *testing.T) {
			configHTML, err := formatReportConfigDetails(tc.snapshot, 50, 30)
			s.Require().NoError(err)
			expectedHTML := strings.ReplaceAll(tc.expectedHTML, "\n", "")
			expectedHTML = strings.ReplaceAll(expectedHTML, "\t", "")
			s.Require().Equal(expectedHTML, configHTML)
		})
	}
}

func (s *EmailFormatterTestSuite) configDetailsTestCases() []configDetailsTestCase {
	cases := []configDetailsTestCase{
		{
			desc: "All severities, image types, fixabilities; Cves since last scheduled report",
			snapshot: func() *storage.ReportSnapshot {
				snap := fixtures.GetReportSnapshot()
				snap.GetVulnReportFilters().CvesSince = &storage.VulnerabilityReportFilters_SinceLastSentScheduledReport{
					SinceLastSentScheduledReport: true,
				}
				return snap
			}(),
			expectedHTML: `<div>
						<div style="padding: 0 0 10px 0">
							<span style="font-weight: bold; margin-right: 10px">
								Config name: </span>
							<span>App Team 1 Report</span>
						</div>
						<div style="padding: 0 0 10px 0">
							<span style="font-weight: bold; margin-right: 10px">
								Number of CVEs found: </span>
							<span>50 in Deployed images, 30 in Watched images</span>
						</div>
						<div style="padding: 0 0 10px 0">
							<span style="font-weight: bold; margin-right: 10px">
								CVE severity: </span>
							<span>Critical, Important, Moderate, Low</span>
						</div>
						<div style="padding: 0 0 10px 0">
							<span style="font-weight: bold; margin-right: 10px">
								CVE status: </span>
							<span>Fixable, Not fixable</span>
						</div>
						<div style="padding: 0 0 10px 0">
							<span style="font-weight: bold; margin-right: 10px">
								Report scope: </span>
							<span>collection-1</span>
						</div>
						<div style="padding: 0 0 10px 0">
							<span style="font-weight: bold; margin-right: 10px">
								Image type: </span>
							<span>Deployed images, Watched images</span>
						</div>
						<div style="padding: 0 0 10px 0">
							<span style="font-weight: bold; margin-right: 10px">
								CVEs discovered since: </span>
							<span>Last successful scheduled report</span>
						</div>
					</div>`,
		},
		{
			desc:     "All severities, image types, fixabilities; Cves since All time",
			snapshot: fixtures.GetReportSnapshot(),
			expectedHTML: `<div>
						<div style="padding: 0 0 10px 0">
							<span style="font-weight: bold; margin-right: 10px">
								Config name: </span>
							<span>App Team 1 Report</span>
						</div>
						<div style="padding: 0 0 10px 0">
							<span style="font-weight: bold; margin-right: 10px">
								Number of CVEs found: </span>
							<span>50 in Deployed images, 30 in Watched images</span>
						</div>
						<div style="padding: 0 0 10px 0">
							<span style="font-weight: bold; margin-right: 10px">
								CVE severity: </span>
							<span>Critical, Important, Moderate, Low</span>
						</div>
						<div style="padding: 0 0 10px 0">
							<span style="font-weight: bold; margin-right: 10px">
								CVE status: </span>
							<span>Fixable, Not fixable</span>
						</div>
						<div style="padding: 0 0 10px 0">
							<span style="font-weight: bold; margin-right: 10px">
								Report scope: </span>
							<span>collection-1</span>
						</div>
						<div style="padding: 0 0 10px 0">
							<span style="font-weight: bold; margin-right: 10px">
								Image type: </span>
							<span>Deployed images, Watched images</span>
						</div>
						<div style="padding: 0 0 10px 0">
							<span style="font-weight: bold; margin-right: 10px">
								CVEs discovered since: </span>
							<span>All time</span>
						</div>
					</div>`,
		},
		{
			desc: "Critical severity, fixable CVEs, Deployed Images; Cves since custom date",
			snapshot: func() *storage.ReportSnapshot {
				snap := fixtures.GetReportSnapshot()
				snap.GetVulnReportFilters().Severities = []storage.VulnerabilitySeverity{
					storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
				}
				snap.GetVulnReportFilters().Fixability = storage.VulnerabilityReportFilters_FIXABLE
				snap.GetVulnReportFilters().ImageTypes = []storage.VulnerabilityReportFilters_ImageType{
					storage.VulnerabilityReportFilters_DEPLOYED,
				}
				dateTs, err := protocompat.ConvertTimeToTimestampOrError(timeutil.MustParse("2006-01-02 15:04:05", "2023-01-20 22:42:02"))
				s.Require().NoError(err)
				snap.GetVulnReportFilters().CvesSince = &storage.VulnerabilityReportFilters_SinceStartDate{
					SinceStartDate: dateTs,
				}
				return snap
			}(),
			expectedHTML: `<div>
						<div style="padding: 0 0 10px 0">
							<span style="font-weight: bold; margin-right: 10px">
								Config name: </span>
							<span>App Team 1 Report</span>
						</div>
						<div style="padding: 0 0 10px 0">
							<span style="font-weight: bold; margin-right: 10px">
								Number of CVEs found: </span>
							<span>50 in Deployed images, 30 in Watched images</span>
						</div>
						<div style="padding: 0 0 10px 0">
							<span style="font-weight: bold; margin-right: 10px">
								CVE severity: </span>
							<span>Critical</span>
						</div>
						<div style="padding: 0 0 10px 0">
							<span style="font-weight: bold; margin-right: 10px">
								CVE status: </span>
							<span>Fixable</span>
						</div>
						<div style="padding: 0 0 10px 0">
							<span style="font-weight: bold; margin-right: 10px">
								Report scope: </span>
							<span>collection-1</span>
						</div>
						<div style="padding: 0 0 10px 0">
							<span style="font-weight: bold; margin-right: 10px">
								Image type: </span>
							<span>Deployed images</span>
						</div>
						<div style="padding: 0 0 10px 0">
							<span style="font-weight: bold; margin-right: 10px">
								CVEs discovered since: </span>
							<span>January 20, 2023</span>
						</div>
					</div>`,
		},
	}
	cases = append(cases, s.entityScopeTestCases()...)
	return cases
}

func (s *EmailFormatterTestSuite) entityScopeTestCases() []configDetailsTestCase {
	return []configDetailsTestCase{
		{
			desc: "Entity scope with filter query cvss > 7",
			snapshot: func() *storage.ReportSnapshot {
				snap := fixtures.GetReportSnapshot()
				snap.Collection = nil
				snap.ResourceScope = &storage.ResourceScope{
					ScopeReference: &storage.ResourceScope_EntityScope{
						EntityScope: &storage.EntityScope{
							Rules: []*storage.EntityScopeRule{
								{
									Entity: storage.EntityType_ENTITY_TYPE_NAMESPACE,
									Field:  storage.EntityField_FIELD_NAME,
									Values: []*storage.RuleValue{
										{Value: "production"},
										{Value: "staging"},
									},
								},
								{
									Entity: storage.EntityType_ENTITY_TYPE_CLUSTER,
									Field:  storage.EntityField_FIELD_NAME,
									Values: []*storage.RuleValue{
										{Value: "main-cluster"},
									},
								},
							},
						},
					},
				}
				snap.GetVulnReportFilters().Query = "CVSS > 7"
				return snap
			}(),
			expectedHTML: `<div>
						<div style="padding: 0 0 10px 0">
							<span style="font-weight: bold; margin-right: 10px">
								Config name: </span>
							<span>App Team 1 Report</span>
						</div>
						<div style="padding: 0 0 10px 0">
							<span style="font-weight: bold; margin-right: 10px">
								Number of CVEs found: </span>
							<span>50 in Deployed images, 30 in Watched images</span>
						</div>
						<div style="padding: 0 0 10px 0">
							<span style="font-weight: bold; margin-right: 10px">
								Filter: </span>
							<span>CVSS > 7</span>
						</div>
						<div style="padding: 0 0 10px 0">
							<span style="font-weight: bold; margin-right: 10px">
								Report scope: </span>
							<span>Namespace Name: production, staging, Cluster Name: main-cluster</span>
						</div>
						<div style="padding: 0 0 10px 0">
							<span style="font-weight: bold; margin-right: 10px">
								Image type: </span>
							<span>Deployed images, Watched images</span>
						</div>
						<div style="padding: 0 0 10px 0">
							<span style="font-weight: bold; margin-right: 10px">
								CVEs discovered since: </span>
							<span>All time</span>
						</div>
					</div>`,
		},
	}
}
