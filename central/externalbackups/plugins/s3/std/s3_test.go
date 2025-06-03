package s3

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

func TestValidate(t *testing.T) {
	for name, tc := range map[string]testdata.PluginConfigTestCase{
		"Valid config returns no error": {
			InputConfig:             testdata.GetValidS3ConfigUsingIAM(t),
			ExpectedValidationError: nil,
		},
		"Wrong config type returns an error": {
			InputConfig:             testdata.GetValidS3CompatibleConfigPathStyleBucket(t),
			ExpectedValidationError: errors.New("S3 configuration required"),
		},
		"S3 config missing Bucket returns an error": {
			InputConfig:             testdata.GetBrokenS3ConfigNoBucket(t),
			ExpectedValidationError: errors.New("S3 Validation error: Bucket must be specified"),
		},
		"S3 config without IAM nor access key ID returns an error": {
			InputConfig:             testdata.GetBrokenS3ConfigNoIAMNoAccessKeyID(t),
			ExpectedValidationError: errors.New("S3 Validation error: Access Key ID must be specified"),
		},
		"S3 config without IAM nor access secret returns an error": {
			InputConfig:             testdata.GetBrokenS3ConfigNoIAMNoAccessSecret(t),
			ExpectedValidationError: errors.New("S3 Validation error: Secret Access Key must be specified"),
		},
		"S3 config without IAM nor access data returns an error": {
			InputConfig:             testdata.GetBrokenS3ConfigNoIAMNoAccessData(t),
			ExpectedValidationError: errors.New("S3 Validation errors: [Access Key ID must be specified, Secret Access Key must be specified]"),
		},
		"S3 config using IAM with access key ID returns an error": {
			InputConfig:             testdata.GetBrokenS3ConfigUsingIAMAndAccessKeyID(t),
			ExpectedValidationError: errors.New("S3 Validation error: IAM and access/secret key use are mutually exclusive. Only specify one"),
		},
		"S3 config using IAM with access secret returns an error": {
			InputConfig:             testdata.GetBrokenS3ConfigUsingIAMAndAccessSecret(t),
			ExpectedValidationError: errors.New("S3 Validation error: IAM and access/secret key use are mutually exclusive. Only specify one"),
		},
		"S3 config missing region returns an error": {
			InputConfig:             testdata.GetBrokenS3ConfigNoRegion(t),
			ExpectedValidationError: errors.New("S3 Validation error: Region must be specified"),
		},
	} {
		t.Run(name, func(it *testing.T) {
			wrapper := &s3configWrapper{integration: tc.InputConfig}
			err := wrapper.Validate()
			if tc.ExpectedValidationError == nil {
				assert.NoError(it, err)
			} else {
				assert.ErrorContains(it, err, tc.ExpectedValidationError.Error())
			}
		})
	}
}
