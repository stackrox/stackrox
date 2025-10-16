package common

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
)

// GetTestReportConfigsV1 returns v1 configs for testing
func GetTestReportConfigsV1(_ *testing.T, notifierID, collectionID string) []*storage.ReportConfiguration {
	v1Config1 := fixtures.GetValidReportConfigWithMultipleNotifiersV1()
	v1Config1.SetId("")
	v1Config1.SetName("Report Config - 1")
	v1Config1.GetEmailConfig().SetNotifierId(notifierID)
	v1Config1.SetScopeId(collectionID)

	v1Config2 := fixtures.GetValidReportConfigWithMultipleNotifiersV1()
	v1Config2.SetId("")
	v1Config2.SetName("Report Config - 2")
	v1Config2.GetEmailConfig().SetNotifierId(notifierID)
	v1Config2.SetScopeId(collectionID)

	return []*storage.ReportConfiguration{v1Config1, v1Config2}
}

// GetTestReportConfigsV2 returns v2 configs for testing
func GetTestReportConfigsV2(_ *testing.T, notifierID, collectionID string) []*storage.ReportConfiguration {
	v2Config1 := fixtures.GetValidReportConfigWithMultipleNotifiersV2()
	v2Config1.SetId("")
	v2Config1.SetName("Report Config - 1")
	for _, notifierConf := range v2Config1.GetNotifiers() {
		notifierConf.SetId(notifierID)
	}
	v2Config1.GetResourceScope().SetCollectionId(collectionID)

	v2Config2 := fixtures.GetValidReportConfigWithMultipleNotifiersV2()
	v2Config2.SetId("")
	v2Config2.SetName("Report Config - 2")
	for _, notifierConf := range v2Config2.GetNotifiers() {
		notifierConf.SetId(notifierID)
	}
	v2Config2.GetResourceScope().SetCollectionId(collectionID)

	return []*storage.ReportConfiguration{v2Config1, v2Config2}
}
