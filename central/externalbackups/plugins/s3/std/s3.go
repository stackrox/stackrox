package s3

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/externalbackups/plugins"
	s3common "github.com/stackrox/rox/central/externalbackups/plugins/s3/common"
	"github.com/stackrox/rox/central/externalbackups/plugins/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errorhelpers"
)

type s3configWrapper struct {
	integration *storage.ExternalBackup
}

func (c *s3configWrapper) GetUrlStyle() storage.S3URLStyle {
	return storage.S3URLStyle_S3_URL_STYLE_UNSPECIFIED
}

func (c *s3configWrapper) GetEndpoint() string {
	return c.integration.GetS3().GetEndpoint()
}

func (c *s3configWrapper) GetValidatedEndpoint() (string, error) {
	return c.GetEndpoint(), nil
}

func (c *s3configWrapper) GetRegion() string {
	return c.integration.GetS3().GetRegion()
}

func (c *s3configWrapper) GetBucket() string {
	return c.integration.GetS3().GetBucket()
}

func (c *s3configWrapper) GetObjectPrefix() string {
	return c.integration.GetS3().GetObjectPrefix()
}

func (c *s3configWrapper) GetUseIam() bool {
	return c.integration.GetS3().GetUseIam()
}

func (c *s3configWrapper) GetAccessKeyId() string {
	return c.integration.GetS3().GetAccessKeyId()
}

func (c *s3configWrapper) GetSecretAccessKey() string {
	return c.integration.GetS3().GetSecretAccessKey()
}

func (c *s3configWrapper) GetName() string {
	return c.integration.GetName()
}

func (c *s3configWrapper) GetErrorCode() string {
	return types.S3Type
}

func (c *s3configWrapper) GetBackupsToKeep() int32 {
	return c.integration.GetBackupsToKeep()
}

func (c *s3configWrapper) Validate() error {
	_, ok := c.integration.Config.(*storage.ExternalBackup_S3)
	if !ok {
		return errors.New("S3 configuration required")
	}

	errorList := errorhelpers.NewErrorList("S3 Validation")
	if c.GetBucket() == "" {
		errorList.AddString("Bucket must be specified")
	}
	if !c.GetUseIam() {
		if c.GetAccessKeyId() == "" {
			errorList.AddString("Access Key ID must be specified")
		}
		if c.GetSecretAccessKey() == "" {
			errorList.AddString("Secret Access Key must be specified")
		}
	} else if c.GetAccessKeyId() != "" || c.GetSecretAccessKey() != "" {
		errorList.AddStrings("IAM and access/secret key use are mutually exclusive. Only specify one")
	}
	if c.GetRegion() == "" {
		errorList.AddString("Region must be specified")
	}

	return errorList.ToError()
}

func init() {
	plugins.Add(types.S3Type, func(backup *storage.ExternalBackup) (types.ExternalBackup, error) {
		return s3common.NewS3Client(&s3configWrapper{integration: backup})
	})
}
