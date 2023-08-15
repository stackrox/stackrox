package common

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
)

// GetTestReportConfigsV1 returns v1 configs for testing
func GetTestReportConfigsV1(_ *testing.T, notifierId, collectionId string) []*storage.ReportConfiguration {
	v1Config1 := fixtures.GetValidReportConfigWithMultipleNotifiersV1()
	v1Config1.Id = ""
	v1Config1.Name = "Report Config - 1"
	v1Config1.GetEmailConfig().NotifierId = notifierId
	v1Config1.ScopeId = collectionId

	v1Config2 := fixtures.GetValidReportConfigWithMultipleNotifiersV1()
	v1Config2.Id = ""
	v1Config2.Name = "Report Config - 2"
	v1Config2.GetEmailConfig().NotifierId = notifierId
	v1Config2.ScopeId = collectionId

	return []*storage.ReportConfiguration{v1Config1, v1Config2}
}

// GetTestReportConfigsV2 returns v2 configs for testing
func GetTestReportConfigsV2(_ *testing.T, notifierId, collectionId string) []*storage.ReportConfiguration {
	v2Config1 := fixtures.GetValidReportConfigWithMultipleNotifiersV2()
	v2Config1.Id = ""
	v2Config1.Name = "Report Config - 1"
	for _, notifierConf := range v2Config1.GetNotifiers() {
		notifierConf.Ref = &storage.NotifierConfiguration_Id{
			Id: notifierId,
		}
	}
	v2Config1.ResourceScope.ScopeReference = &storage.ResourceScope_CollectionId{
		CollectionId: collectionId,
	}

	v2Config2 := fixtures.GetValidReportConfigWithMultipleNotifiersV2()
	v2Config2.Id = ""
	v2Config2.Name = "Report Config - 2"
	for _, notifierConf := range v2Config2.GetNotifiers() {
		notifierConf.Ref = &storage.NotifierConfiguration_Id{
			Id: notifierId,
		}
	}
	v2Config2.ResourceScope.ScopeReference = &storage.ResourceScope_CollectionId{
		CollectionId: collectionId,
	}

	return []*storage.ReportConfiguration{v2Config1, v2Config2}
}
