package testdata

import (
	"testing"

	"github.com/stackrox/rox/central/externalbackups/plugins/s3/common"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

const (
	// S3Endpoint is a test endpoint value
	S3Endpoint = "s3.example.com"
	// S3CompatibleEndpoint is a test endpoint value
	S3CompatibleEndpoint = "s3compatible.example.com"

	// TestRegion is a test region value
	TestRegion = "uk-west-1"

	// S3Bucket is a test bucket value
	S3Bucket = "ValidS3Bucket"
	// S3CompatibleBucket is a test bucket value
	S3CompatibleBucket = "ValidS3CompatibleBucket"

	// S3ObjectPrefix is a test object key prefix value
	S3ObjectPrefix = "valid/s3"
	// S3CompatibleObjectPrefix is a test object key prefix value
	S3CompatibleObjectPrefix = "valid/s3compatible"

	// TestAccessKeyID is a test access key ID value
	TestAccessKeyID = "SomeKeyID"
	// TestSecretAccessKey is a test value for the secret access key field
	TestSecretAccessKey = "In a hole in the ground there lived a hobbit."

	// S3IntegrationName is a test value for the integration name
	S3IntegrationName = "Valid S3 backup plugin configuration"
	// S3CompatibleIntegrationName is a test value for the integration name
	S3CompatibleIntegrationName = "Valid S3 compatible backup plugin configuration"

	// TestKeepTwoBackups is a test value for the BackupsToKeep field
	TestKeepTwoBackups = int32(2)
	// TestKeepThreeBackups is a test value for the BackupsToKeep field
	TestKeepThreeBackups = int32(3)
)

// PluginConfigTestCase is a structure that contains the input
// and expected outputs for config wrapper tests.
type PluginConfigTestCase struct {
	InputConfig             *storage.ExternalBackup
	ExpectedURLStyle        storage.S3URLStyle
	ExpectedEndpoint        string
	ExpectedRegion          string
	ExpectedBucket          string
	ExpectedObjectPrefix    string
	ExpectedUseIam          bool
	ExpectedAccessKeyID     string
	ExpectedSecretAccessKey string
	ExpectedName            string
	ExpectedPluginType      string
	ExpectedBackupsToKeep   int32
	ExpectedValidationError error
}

// TestAccessors runs the basic accessor tests configured in the input test cases
// against the config wrappers built by the input factory.
func TestAccessors(
	t *testing.T,
	wrapperFactory func(backup *storage.ExternalBackup) s3common.ConfigWrapper,
	testCases map[string]PluginConfigTestCase,
) {
	t.Run("TestGetUrlStyle", func(it *testing.T) {
		for name, tc := range testCases {
			it.Run(name, func(iit *testing.T) {
				wrapper := wrapperFactory(tc.InputConfig)
				assert.Equal(iit, tc.ExpectedURLStyle, wrapper.GetUrlStyle())
			})
		}
	})
	t.Run("TestGetEndpoint", func(it *testing.T) {
		for name, tc := range testCases {
			it.Run(name, func(iit *testing.T) {
				wrapper := wrapperFactory(tc.InputConfig)
				assert.Equal(iit, tc.ExpectedEndpoint, wrapper.GetEndpoint())
			})
		}
	})
	t.Run("TestGetRegion", func(it *testing.T) {
		for name, tc := range testCases {
			it.Run(name, func(iit *testing.T) {
				wrapper := wrapperFactory(tc.InputConfig)
				assert.Equal(iit, tc.ExpectedRegion, wrapper.GetRegion())
			})
		}
	})
	t.Run("TestGetBucket", func(it *testing.T) {
		for name, tc := range testCases {
			it.Run(name, func(iit *testing.T) {
				wrapper := wrapperFactory(tc.InputConfig)
				assert.Equal(iit, tc.ExpectedBucket, wrapper.GetBucket())
			})
		}
	})
	t.Run("TestGetObjectPrefix", func(it *testing.T) {
		for name, tc := range testCases {
			it.Run(name, func(iit *testing.T) {
				wrapper := wrapperFactory(tc.InputConfig)
				assert.Equal(iit, tc.ExpectedObjectPrefix, wrapper.GetObjectPrefix())
			})
		}
	})
	t.Run("TestGetUseIam", func(it *testing.T) {
		for name, tc := range testCases {
			it.Run(name, func(iit *testing.T) {
				wrapper := wrapperFactory(tc.InputConfig)
				assert.Equal(iit, tc.ExpectedUseIam, wrapper.GetUseIam())
			})
		}
	})
	t.Run("TestGetAccessKeyId", func(it *testing.T) {
		for name, tc := range testCases {
			it.Run(name, func(iit *testing.T) {
				wrapper := wrapperFactory(tc.InputConfig)
				assert.Equal(iit, tc.ExpectedAccessKeyID, wrapper.GetAccessKeyId())
			})
		}
	})
	t.Run("TestGetSecretAccessKey", func(it *testing.T) {
		for name, tc := range testCases {
			it.Run(name, func(iit *testing.T) {
				wrapper := wrapperFactory(tc.InputConfig)
				assert.Equal(iit, tc.ExpectedSecretAccessKey, wrapper.GetSecretAccessKey())
			})
		}
	})
	t.Run("TestGetName", func(it *testing.T) {
		for name, tc := range testCases {
			it.Run(name, func(iit *testing.T) {
				wrapper := wrapperFactory(tc.InputConfig)
				assert.Equal(iit, tc.ExpectedName, wrapper.GetName())
			})
		}
	})
	t.Run("TestGetPluginType", func(it *testing.T) {
		for name, tc := range testCases {
			it.Run(name, func(iit *testing.T) {
				wrapper := wrapperFactory(tc.InputConfig)
				assert.Equal(iit, tc.ExpectedPluginType, wrapper.GetPluginType())
			})
		}
	})
	t.Run("TestGetBackupsToKeep", func(it *testing.T) {
		for name, tc := range testCases {
			it.Run(name, func(iit *testing.T) {
				wrapper := wrapperFactory(tc.InputConfig)
				assert.Equal(iit, tc.ExpectedBackupsToKeep, wrapper.GetBackupsToKeep())
			})
		}
	})
}

// GetValidS3ConfigNoIAM returns a proto container for s3 and compatible
// config wrapper tests.
func GetValidS3ConfigNoIAM(_ testing.TB) *storage.ExternalBackup {
	return &storage.ExternalBackup{
		Id:            "ValidS3ConfigID",
		Name:          S3IntegrationName,
		Type:          "S3",
		BackupsToKeep: TestKeepTwoBackups,
		Config: &storage.ExternalBackup_S3{
			S3: &storage.S3Config{
				Bucket:          S3Bucket,
				UseIam:          false,
				AccessKeyId:     TestAccessKeyID,
				SecretAccessKey: TestSecretAccessKey,
				Region:          TestRegion,
				ObjectPrefix:    S3ObjectPrefix,
				Endpoint:        S3Endpoint,
			},
		},
	}
}

// GetValidS3ConfigUsingIAM returns a proto container for s3 and compatible
// config wrapper tests.
func GetValidS3ConfigUsingIAM(_ testing.TB) *storage.ExternalBackup {
	return &storage.ExternalBackup{
		Id:            "ValidS3ConfigID",
		Name:          S3IntegrationName,
		Type:          "S3",
		BackupsToKeep: TestKeepThreeBackups,
		Config: &storage.ExternalBackup_S3{
			S3: &storage.S3Config{
				Bucket:       S3Bucket,
				UseIam:       true,
				Region:       TestRegion,
				ObjectPrefix: S3ObjectPrefix,
				Endpoint:     S3Endpoint,
			},
		},
	}
}

// GetValidS3CompatibleConfigVirtualHosted returns a proto container for s3 and compatible
// config wrapper tests.
func GetValidS3CompatibleConfigVirtualHosted(_ testing.TB) *storage.ExternalBackup {
	return &storage.ExternalBackup{
		Id:            "ValidS3CompatibleConfigID",
		Name:          S3CompatibleIntegrationName,
		Type:          "S3Compatible",
		BackupsToKeep: TestKeepTwoBackups,
		Config: &storage.ExternalBackup_S3Compatible{
			S3Compatible: &storage.S3Compatible{
				Bucket:          S3CompatibleBucket,
				AccessKeyId:     TestAccessKeyID,
				SecretAccessKey: TestSecretAccessKey,
				Region:          TestRegion,
				ObjectPrefix:    S3CompatibleObjectPrefix,
				Endpoint:        S3CompatibleEndpoint,
				UrlStyle:        storage.S3URLStyle_S3_URL_STYLE_VIRTUAL_HOSTED,
			},
		},
	}
}

// GetValidS3CompatibleConfigPathStyleBucket returns a proto container for s3 and compatible
// config wrapper tests.
func GetValidS3CompatibleConfigPathStyleBucket(_ testing.TB) *storage.ExternalBackup {
	return &storage.ExternalBackup{
		Id:            "ValidS3CompatibleConfigID",
		Name:          S3CompatibleIntegrationName,
		Type:          "S3Compatible",
		BackupsToKeep: TestKeepThreeBackups,
		Config: &storage.ExternalBackup_S3Compatible{
			S3Compatible: &storage.S3Compatible{
				Bucket:          S3CompatibleBucket,
				AccessKeyId:     TestAccessKeyID,
				SecretAccessKey: TestSecretAccessKey,
				Region:          TestRegion,
				ObjectPrefix:    S3CompatibleObjectPrefix,
				Endpoint:        S3CompatibleEndpoint,
				UrlStyle:        storage.S3URLStyle_S3_URL_STYLE_PATH,
			},
		},
	}
}
