package reportconfigurations

import (
	"testing"

	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stretchr/testify/assert"
)

func TestConvertV2ReportConfigurationToProto(t *testing.T) {
	var cases = []struct {
		testname        string
		reportConfigGen func() *v2.ReportConfiguration
		resultGen       func() *storage.ReportConfiguration
	}{
		{
			testname: "Report config with notifiers",
			reportConfigGen: func() *v2.ReportConfiguration {
				return fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
			},
			resultGen: func() *storage.ReportConfiguration {
				return fixtures.GetValidReportConfigWithMultipleNotifiers()
			},
		},
		{
			testname: "Report config without notifiers",
			reportConfigGen: func() *v2.ReportConfiguration {
				ret := fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
				ret.Notifiers = nil
				return ret
			},
			resultGen: func() *storage.ReportConfiguration {
				ret := fixtures.GetValidReportConfigWithMultipleNotifiers()
				ret.Notifiers = nil
				return ret
			},
		},
		{
			testname: "Report config without schedule",
			reportConfigGen: func() *v2.ReportConfiguration {
				ret := fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
				ret.Schedule = nil
				return ret
			},
			resultGen: func() *storage.ReportConfiguration {
				ret := fixtures.GetValidReportConfigWithMultipleNotifiers()
				ret.Schedule = nil
				return ret
			},
		},
		{
			testname: "Report config without filter",
			reportConfigGen: func() *v2.ReportConfiguration {
				ret := fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
				ret.Filter = nil
				return ret
			},
			resultGen: func() *storage.ReportConfiguration {
				ret := fixtures.GetValidReportConfigWithMultipleNotifiers()
				ret.Filter = nil
				return ret
			},
		},
		{
			testname: "Report config without resource scope",
			reportConfigGen: func() *v2.ReportConfiguration {
				ret := fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
				ret.ResourceScope = nil
				return ret
			},
			resultGen: func() *storage.ReportConfiguration {
				ret := fixtures.GetValidReportConfigWithMultipleNotifiers()
				ret.ResourceScope = nil
				return ret
			},
		},
		{
			testname: "Report config without CvesSince in filter",
			reportConfigGen: func() *v2.ReportConfiguration {
				ret := fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
				ret.GetVulnReportFilters().CvesSince = nil
				return ret
			},
			resultGen: func() *storage.ReportConfiguration {
				ret := fixtures.GetValidReportConfigWithMultipleNotifiers()
				ret.GetVulnReportFilters().CvesSince = nil
				return ret
			},
		},
		{
			testname: "Report config without scope reference in ResourceScope",
			reportConfigGen: func() *v2.ReportConfiguration {
				ret := fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
				ret.ResourceScope.ScopeReference = nil
				return ret
			},
			resultGen: func() *storage.ReportConfiguration {
				ret := fixtures.GetValidReportConfigWithMultipleNotifiers()
				ret.ResourceScope.ScopeReference = nil
				return ret
			},
		},
	}

	for _, c := range cases {
		t.Run(c.testname, func(t *testing.T) {
			reportConfig := c.reportConfigGen()
			expected := c.resultGen()
			converted := ConvertV2ReportConfigurationToProto(reportConfig)
			assert.Equal(t, expected, converted)
		})
	}
}

func TestConvertProtoReportConfigurationToV2(t *testing.T) {
	var cases = []struct {
		testname        string
		reportConfigGen func() *storage.ReportConfiguration
		resultGen       func() *v2.ReportConfiguration
	}{
		{
			testname: "Report config with notifiers",
			reportConfigGen: func() *storage.ReportConfiguration {
				return fixtures.GetValidReportConfigWithMultipleNotifiers()
			},
			resultGen: func() *v2.ReportConfiguration {
				return fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
			},
		},
		{
			testname: "Report config without notifiers",
			reportConfigGen: func() *storage.ReportConfiguration {
				ret := fixtures.GetValidReportConfigWithMultipleNotifiers()
				ret.Notifiers = nil
				return ret
			},
			resultGen: func() *v2.ReportConfiguration {
				ret := fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
				ret.Notifiers = nil
				return ret
			},
		},
		{
			testname: "Report config without schedule",
			reportConfigGen: func() *storage.ReportConfiguration {
				ret := fixtures.GetValidReportConfigWithMultipleNotifiers()
				ret.Schedule = nil
				return ret
			},
			resultGen: func() *v2.ReportConfiguration {
				ret := fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
				ret.Schedule = nil
				return ret
			},
		},
		{
			testname: "Report config without filter",
			reportConfigGen: func() *storage.ReportConfiguration {
				ret := fixtures.GetValidReportConfigWithMultipleNotifiers()
				ret.Filter = nil
				return ret
			},
			resultGen: func() *v2.ReportConfiguration {
				ret := fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
				ret.Filter = nil
				return ret
			},
		},
		{
			testname: "Report config without resource scope",
			reportConfigGen: func() *storage.ReportConfiguration {
				ret := fixtures.GetValidReportConfigWithMultipleNotifiers()
				ret.ResourceScope = nil
				return ret
			},
			resultGen: func() *v2.ReportConfiguration {
				ret := fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
				ret.ResourceScope = nil
				return ret
			},
		},
		{
			testname: "Report config without CvesSince in filter",
			reportConfigGen: func() *storage.ReportConfiguration {
				ret := fixtures.GetValidReportConfigWithMultipleNotifiers()
				ret.GetVulnReportFilters().CvesSince = nil
				return ret
			},
			resultGen: func() *v2.ReportConfiguration {
				ret := fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
				ret.GetVulnReportFilters().CvesSince = nil
				return ret
			},
		},
		{
			testname: "Report config without scope reference in ResourceScope",
			reportConfigGen: func() *storage.ReportConfiguration {
				ret := fixtures.GetValidReportConfigWithMultipleNotifiers()
				ret.ResourceScope.ScopeReference = nil
				return ret
			},
			resultGen: func() *v2.ReportConfiguration {
				ret := fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
				ret.ResourceScope.ScopeReference = nil
				return ret
			},
		},
	}

	for _, c := range cases {
		t.Run(c.testname, func(t *testing.T) {
			reportConfig := c.reportConfigGen()
			expected := c.resultGen()
			converted := ConvertProtoReportConfigurationToV2(reportConfig)
			assert.Equal(t, expected, converted)
		})
	}
}
