package s3

import (
	"testing"

	s3common "github.com/stackrox/rox/central/externalbackups/plugins/s3/common"
	"github.com/stackrox/rox/central/externalbackups/plugins/s3/testdata"
	"github.com/stackrox/rox/central/externalbackups/plugins/types"
	"github.com/stackrox/rox/generated/storage"
)

func getAccessorTestCases(t *testing.T) map[string]testdata.PluginConfigTestCase {
	return map[string]testdata.PluginConfigTestCase{
		"valid s3 config without IAM configured": {
			InputConfig:             testdata.GetValidS3ConfigNoIAM(t),
			ExpectedURLStyle:        storage.S3URLStyle_S3_URL_STYLE_UNSPECIFIED,
			ExpectedEndpoint:        testdata.S3Endpoint,
			ExpectedRegion:          testdata.TestRegion,
			ExpectedBucket:          testdata.S3Bucket,
			ExpectedObjectPrefix:    testdata.S3ObjectPrefix,
			ExpectedUseIam:          false,
			ExpectedAccessKeyID:     testdata.TestAccessKeyID,
			ExpectedSecretAccessKey: testdata.TestSecretAccessKey,
			ExpectedName:            testdata.S3IntegrationName,
			ExpectedPluginType:      types.S3Type,
			ExpectedBackupsToKeep:   testdata.TestKeepTwoBackups,
			ExpectedValidationError: nil,
		},
		"valid s3 config with IAM configured": {
			InputConfig:             testdata.GetValidS3ConfigUsingIAM(t),
			ExpectedURLStyle:        storage.S3URLStyle_S3_URL_STYLE_UNSPECIFIED,
			ExpectedEndpoint:        testdata.S3Endpoint,
			ExpectedRegion:          testdata.TestRegion,
			ExpectedBucket:          testdata.S3Bucket,
			ExpectedObjectPrefix:    testdata.S3ObjectPrefix,
			ExpectedUseIam:          true,
			ExpectedAccessKeyID:     "",
			ExpectedSecretAccessKey: "",
			ExpectedName:            testdata.S3IntegrationName,
			ExpectedPluginType:      types.S3Type,
			ExpectedBackupsToKeep:   testdata.TestKeepThreeBackups,
		},
		"s3 compatible config with virtual-hosted bucket type": {
			InputConfig:      testdata.GetValidS3CompatibleConfigVirtualHosted(t),
			ExpectedURLStyle: storage.S3URLStyle_S3_URL_STYLE_UNSPECIFIED,
			// Empty values are expected for the fields from the Config subtree
			// (Endpoint, Region, Bucket, Object key prefix, UseIAM, Access key data)
			// The configured values are expected for the fields at the root of
			// the storage.ExternalBackup structure
			ExpectedName:          testdata.S3CompatibleIntegrationName,
			ExpectedPluginType:    types.S3Type,
			ExpectedBackupsToKeep: testdata.TestKeepTwoBackups,
		},
		"s3 compatible config with path-type bucket type": {
			InputConfig:      testdata.GetValidS3CompatibleConfigPathStyleBucket(t),
			ExpectedURLStyle: storage.S3URLStyle_S3_URL_STYLE_UNSPECIFIED,
			// Empty values are expected for the fields from the Config subtree
			// (Endpoint, Region, Bucket, Object key prefix, UseIAM, Access key data)
			// The configured values are expected for the fields at the root of
			// the storage.ExternalBackup structure
			ExpectedName:          testdata.S3CompatibleIntegrationName,
			ExpectedPluginType:    types.S3Type,
			ExpectedBackupsToKeep: testdata.TestKeepThreeBackups,
		},
	}
}

func TestS3WrapperAccessors(t *testing.T) {
	configWrapperFactory := func(integration *storage.ExternalBackup) s3common.ConfigWrapper {
		return &s3configWrapper{integration: integration}
	}
	testdata.TestAccessors(t, configWrapperFactory, getAccessorTestCases(t))
}
