package s3compatible

import (
	"errors"
	"testing"

	s3common "github.com/stackrox/rox/central/externalbackups/plugins/s3/common"
	"github.com/stackrox/rox/central/externalbackups/plugins/s3/testdata"
	"github.com/stackrox/rox/central/externalbackups/plugins/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func getAccessorTestCases(t *testing.T) map[string]testdata.PluginConfigTestCase {
	return map[string]testdata.PluginConfigTestCase{
		"valid s3 config without IAM configured": {
			InputConfig: testdata.GetValidS3ConfigNoIAM(t),
			// Empty/default values are expected for the fields from the Config subtree
			// (URLStyle, Endpoint, Region, Bucket, Object key prefix, Access key data)
			// The configured values are expected for the fields at the root of
			// the storage.ExternalBackup structure
			ExpectedName:            testdata.S3IntegrationName,
			ExpectedPluginType:      types.S3CompatibleType,
			ExpectedBackupsToKeep:   testdata.TestKeepTwoBackups,
			ExpectedValidationError: nil,
		},
		"valid s3 config with IAM configured": {
			InputConfig: testdata.GetValidS3ConfigUsingIAM(t),
			// Empty/default values are expected for the fields from the Config subtree
			// (URLStyle, Endpoint, Region, Bucket, Object key prefix, Access key data)
			// The configured values are expected for the fields at the root of
			// the storage.ExternalBackup structure
			ExpectedName:          testdata.S3IntegrationName,
			ExpectedPluginType:    types.S3CompatibleType,
			ExpectedBackupsToKeep: testdata.TestKeepThreeBackups,
		},
		"s3 compatible config with virtual-hosted bucket type": {
			InputConfig:             testdata.GetValidS3CompatibleConfigVirtualHosted(t),
			ExpectedURLStyle:        storage.S3URLStyle_S3_URL_STYLE_VIRTUAL_HOSTED,
			ExpectedEndpoint:        testdata.S3CompatibleEndpoint,
			ExpectedRegion:          testdata.TestRegion,
			ExpectedBucket:          testdata.S3CompatibleBucket,
			ExpectedObjectPrefix:    testdata.S3CompatibleObjectPrefix,
			ExpectedUseIam:          false,
			ExpectedAccessKeyID:     testdata.TestAccessKeyID,
			ExpectedSecretAccessKey: testdata.TestSecretAccessKey,
			ExpectedName:            testdata.S3CompatibleIntegrationName,
			ExpectedPluginType:      types.S3CompatibleType,
			ExpectedBackupsToKeep:   testdata.TestKeepTwoBackups,
		},
		"s3 compatible config with path-type bucket type": {
			InputConfig:             testdata.GetValidS3CompatibleConfigPathStyleBucket(t),
			ExpectedURLStyle:        storage.S3URLStyle_S3_URL_STYLE_PATH,
			ExpectedEndpoint:        testdata.S3CompatibleEndpoint,
			ExpectedRegion:          testdata.TestRegion,
			ExpectedBucket:          testdata.S3CompatibleBucket,
			ExpectedObjectPrefix:    testdata.S3CompatibleObjectPrefix,
			ExpectedUseIam:          false,
			ExpectedAccessKeyID:     testdata.TestAccessKeyID,
			ExpectedSecretAccessKey: testdata.TestSecretAccessKey,
			ExpectedName:            testdata.S3CompatibleIntegrationName,
			ExpectedPluginType:      types.S3CompatibleType,
			ExpectedBackupsToKeep:   testdata.TestKeepThreeBackups,
		},
	}
}

func TestS3CompatibleWrapperAccessors(t *testing.T) {
	configWrapperFactory := func(integration *storage.ExternalBackup) s3common.ConfigWrapper {
		return &s3compatibleConfigWrapper{integration: integration}
	}
	testdata.TestAccessors(t, configWrapperFactory, getAccessorTestCases(t))
}

func TestValidate(t *testing.T) {
	for name, tc := range map[string]testdata.PluginConfigTestCase{
		"Valid config returns no error": {
			InputConfig:             testdata.GetValidS3CompatibleConfigPathStyleBucket(t),
			ExpectedValidationError: nil,
		},
		"Wrong config type returns an error": {
			InputConfig:             testdata.GetValidS3ConfigNoIAM(t),
			ExpectedValidationError: errors.New("S3 Compatible configuration required"),
		},
		"S3 compatible config missing access key ID returns an error": {
			InputConfig:             testdata.GetBrokenS3CompatibleConfigNoAccessID(t),
			ExpectedValidationError: errors.New("S3 Compatible Validation error: Access Key ID must be specified"),
		},
		"S3 compatible config missing access secret returns an error": {
			InputConfig:             testdata.GetBrokenS3CompatibleConfigNoAccessSecret(t),
			ExpectedValidationError: errors.New("S3 Compatible Validation error: Secret Access Key must be specified"),
		},
		"S3 compatible config missing access data returns an error": {
			InputConfig:             testdata.GetBrokenS3CompatibleConfigNoAccessData(t),
			ExpectedValidationError: errors.New("S3 Compatible Validation errors: [Access Key ID must be specified, Secret Access Key must be specified]"),
		},
		"S3 compatible config missing region returns an error": {
			InputConfig:             testdata.GetBrokenS3CompatibleConfigNoRegion(t),
			ExpectedValidationError: errors.New("S3 Compatible Validation error: Region must be specified"),
		},
	} {
		t.Run(name, func(it *testing.T) {
			wrapper := &s3compatibleConfigWrapper{integration: tc.InputConfig}
			err := wrapper.Validate()
			if tc.ExpectedValidationError == nil {
				assert.NoError(it, err)
			} else {
				assert.ErrorContains(it, err, tc.ExpectedValidationError.Error())
			}
		})
	}
}
