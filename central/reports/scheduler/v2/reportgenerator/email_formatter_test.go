package reportgenerator

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/timeutil"
	"github.com/stretchr/testify/suite"
)

func TestEmailFormatter(t *testing.T) {
	suite.Run(t, new(EmailFormatterTestSuite))
}

type EmailFormatterTestSuite struct {
	suite.Suite
}

func (s *EmailFormatterTestSuite) TestFormatWorkloadReportEmailBody_CollectionScope() {
	snap := fixtures.GetReportSnapshot()
	reportURL := "https://acs.example.com/main/vulnerabilities/reports/images/configurations/abc123"

	html, err := FormatWorkloadReportEmailBody("Intro text.", snap, 6298, 92, true, reportURL)
	s.Require().NoError(err)

	// Header, title and subtitle.
	s.Contains(html, "Workload CVE Report")
	s.Contains(html, "App Team 1 Report")
	s.Contains(html, `src="cid:logo.png"`)
	// Intro passed through.
	s.Contains(html, "Intro text.")
	// Stat cards with thousands separators.
	s.Contains(html, "6,298")
	s.Contains(html, "92")
	s.Contains(html, "Deployed images")
	s.Contains(html, "Watched images")
	// Severity legend (fixture has all severities).
	s.Contains(html, "CVE severity")
	s.Contains(html, "Critical")
	s.Contains(html, "Important")
	// Details table.
	s.Contains(html, "CVE status")
	s.Contains(html, "Fixable")
	s.Contains(html, "collection-1")
	s.Contains(html, "CVEs discovered in image since")
	s.Contains(html, "All time")
	// Attachment note and console button.
	s.Contains(html, "attached as a CSV/ZIP")
	s.Contains(html, reportURL)
	s.Contains(html, "View report in console")
	// Not the no-vulns messaging.
	s.NotContains(html, "No workload CVEs found")
}

func (s *EmailFormatterTestSuite) TestFormatWorkloadReportEmailBody_NoVulns() {
	snap := fixtures.GetReportSnapshot()

	html, err := FormatWorkloadReportEmailBody("Intro text.", snap, 0, 0, false, "")
	s.Require().NoError(err)

	s.Contains(html, "No workload CVEs found")
	s.Contains(html, "No report file is attached")
	s.NotContains(html, "attached as a CSV/ZIP")
	// No button when the URL is empty.
	s.NotContains(html, "View report in console")
}

func (s *EmailFormatterTestSuite) TestFormatWorkloadReportEmailBody_EntityScope() {
	snap := fixtures.GetReportSnapshot()
	snap.Collection = nil
	snap.ResourceScope = &storage.ResourceScope{
		ScopeReference: &storage.ResourceScope_EntityScope{
			EntityScope: &storage.EntityScope{
				Rules: []*storage.EntityScopeRule{
					{
						Entity: storage.EntityType_ENTITY_TYPE_NAMESPACE,
						Field:  storage.EntityField_FIELD_NAME,
						Values: []*storage.RuleValue{{Value: "production"}, {Value: "staging"}},
					},
				},
			},
		},
	}
	snap.GetVulnReportFilters().Query = "CVSS > 7"

	html, err := FormatWorkloadReportEmailBody("Intro text.", snap, 12, 88, true, "")
	s.Require().NoError(err)

	// Entity scope shows a Filter row (raw query, HTML-escaped) and scope rules,
	// but no severity legend or CVE status row.
	s.Contains(html, "Filter")
	s.Contains(html, "CVSS &gt; 7")
	s.Contains(html, "Namespace Name: production, staging")
	s.NotContains(html, "CVE severity")
	s.NotContains(html, "CVE status")
}

func (s *EmailFormatterTestSuite) TestFormatWorkloadReportEmailBody_EscapesUserValues() {
	snap := fixtures.GetReportSnapshot()
	snap.Name = "<script>alert(1)</script>"

	html, err := FormatWorkloadReportEmailBody("Intro text.", snap, 1, 0, true, "")
	s.Require().NoError(err)

	s.NotContains(html, "<script>alert(1)</script>")
	s.Contains(html, "&lt;script&gt;")
}

func (s *EmailFormatterTestSuite) TestFormatWorkloadReportEmailBody_CustomDate() {
	snap := fixtures.GetReportSnapshot()
	dateTs, err := protocompat.ConvertTimeToTimestampOrError(timeutil.MustParse("2006-01-02 15:04:05", "2023-01-20 22:42:02"))
	s.Require().NoError(err)
	snap.GetVulnReportFilters().CvesSince = &storage.VulnerabilityReportFilters_SinceStartDate{SinceStartDate: dateTs}

	html, err := FormatWorkloadReportEmailBody("Intro text.", snap, 1, 0, true, "")
	s.Require().NoError(err)
	s.Contains(html, "January 20, 2023")
}

func (s *EmailFormatterTestSuite) TestFormatWorkloadReportEmailBody_MissingFilters() {
	snap := &storage.ReportSnapshot{Name: "Broken"}
	_, err := FormatWorkloadReportEmailBody("Intro text.", snap, 1, 0, true, "")
	s.Require().Error(err)
}

func (s *EmailFormatterTestSuite) TestFormatNodeReportEmailBody() {
	snap := &storage.ReportSnapshot{
		Name: "Node Team Report",
		Filter: &storage.ReportSnapshot_NodeVulnReportFilters{
			NodeVulnReportFilters: &storage.NodeVulnerabilityReportFilters{Query: "CVSS > 7"},
		},
		ResourceScope: &storage.ResourceScope{
			ScopeReference: &storage.ResourceScope_EntityScope{
				EntityScope: &storage.EntityScope{
					Rules: []*storage.EntityScopeRule{
						{
							Entity: storage.EntityType_ENTITY_TYPE_CLUSTER,
							Field:  storage.EntityField_FIELD_NAME,
							Values: []*storage.RuleValue{{Value: "prod-us"}, {Value: "prod-eu"}},
						},
					},
				},
			},
		},
	}

	html, err := FormatNodeReportEmailBody("Intro text.", snap, 247, true,
		"https://acs.example.com/main/vulnerabilities/reports/nodes/configurations/abc")
	s.Require().NoError(err)

	s.Contains(html, "Node CVE Report")
	s.Contains(html, "Node Team Report")
	s.Contains(html, `src="cid:logo.png"`)
	s.Contains(html, "247")
	s.Contains(html, "Node CVEs found")
	s.Contains(html, "Filter")
	s.Contains(html, "CVSS &gt; 7")
	s.Contains(html, "Cluster Name: prod-us, prod-eu")
	s.Contains(html, "attached as a CSV/ZIP")
	// Node reports never render a severity legend.
	s.NotContains(html, "CVE severity")
}

func (s *EmailFormatterTestSuite) TestFormatNodeReportEmailBody_NoVulns() {
	snap := &storage.ReportSnapshot{
		Name: "Simple Node Report",
		Filter: &storage.ReportSnapshot_NodeVulnReportFilters{
			NodeVulnReportFilters: &storage.NodeVulnerabilityReportFilters{},
		},
	}

	html, err := FormatNodeReportEmailBody("Intro text.", snap, 0, false, "")
	s.Require().NoError(err)
	s.Contains(html, "No node CVEs found")
	s.Contains(html, "No report file is attached")
}

func (s *EmailFormatterTestSuite) TestFormatNodeReportEmailBody_NilFilters() {
	snap := &storage.ReportSnapshot{Name: "Node Report"}
	_, err := FormatNodeReportEmailBody("Intro text.", snap, 0, false, "")
	s.Require().Error(err)
}

func (s *EmailFormatterTestSuite) TestHumanizeInt() {
	cases := map[string]struct {
		in  int
		out string
	}{
		"zero":      {0, "0"},
		"tens":      {92, "92"},
		"hundreds":  {999, "999"},
		"thousands": {6298, "6,298"},
		"millions":  {1234567, "1,234,567"},
		"exact-1k":  {1000, "1,000"},
	}
	for name, tc := range cases {
		s.Run(name, func() {
			s.Equal(tc.out, humanizeInt(tc.in))
		})
	}
}

func (s *EmailFormatterTestSuite) TestBuildReportURL() {
	s.Equal("https://acs.example.com/main/vulnerabilities/reports/images/configurations/abc",
		BuildWorkloadReportURL("https://acs.example.com", "abc"))
	s.Equal("https://acs.example.com/main/vulnerabilities/reports/nodes/configurations/abc",
		BuildNodeReportURL("https://acs.example.com", "abc"))
	// Missing endpoint or config ID yields an empty link (button omitted).
	s.Empty(BuildWorkloadReportURL("", "abc"))
	s.Empty(BuildWorkloadReportURL("https://acs.example.com", ""))
}
