package s3compatible

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/externalbackups/plugins"
	s3common "github.com/stackrox/rox/central/externalbackups/plugins/s3/common"
	"github.com/stackrox/rox/central/externalbackups/plugins/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errorhelpers"
)

func init() {

	// s3Compatible plugin for the S3 compatible backup integration.
	// As the official AWS S3 is deprecating the path-style bucket addressing, but
	// this style is still used in non-AWS S3 compatible providers, we decided to
	// implement a new plugin for the latter.
	// Having the two plugins allows for a clear separation between official AWS
	// and non-AWS features.
	// Having the S3 compatible plugin separate will also give us more freedom to
	// change to a different package if the aws-sdk decides to drop the path-style
	// option in the future.

	plugins.Add(types.S3CompatibleType, func(backup *storage.ExternalBackup) (types.ExternalBackup, error) {
		return s3common.NewS3Client(&s3compatibleConfigWrapper{integration: backup})
	})
}

type s3compatibleConfigWrapper struct {
	integration *storage.ExternalBackup
}

func (c *s3compatibleConfigWrapper) GetUrlStyle() storage.S3URLStyle {
	return c.integration.GetS3Compatible().GetUrlStyle()
}

func (c *s3compatibleConfigWrapper) GetEndpoint() string {
	return c.integration.GetS3Compatible().GetEndpoint()
}

func (c *s3compatibleConfigWrapper) GetRegion() string {
	return c.integration.GetS3Compatible().GetRegion()
}

func (c *s3compatibleConfigWrapper) GetBucket() string {
	return c.integration.GetS3Compatible().GetBucket()
}

func (c *s3compatibleConfigWrapper) GetObjectPrefix() string {
	return c.integration.GetS3Compatible().GetObjectPrefix()
}

func (c *s3compatibleConfigWrapper) GetUseIam() bool {
	return false
}

func (c *s3compatibleConfigWrapper) GetAccessKeyId() string {
	return c.integration.GetS3Compatible().GetAccessKeyId()
}

func (c *s3compatibleConfigWrapper) GetSecretAccessKey() string {
	return c.integration.GetS3Compatible().GetSecretAccessKey()
}

func (c *s3compatibleConfigWrapper) GetName() string {
	return c.integration.GetName()
}

func (c *s3compatibleConfigWrapper) GetPluginType() string { return types.S3CompatibleType }

func (c *s3compatibleConfigWrapper) GetBackupsToKeep() int32 {
	return c.integration.GetBackupsToKeep()
}

func (c *s3compatibleConfigWrapper) Validate() error {
	cfg := c.integration.GetS3Compatible()
	if cfg == nil {
		return errors.New("S3 Compatible configuration required")
	}
	errorList := errorhelpers.NewErrorList("S3 Compatible Validation")
	if c.GetAccessKeyId() == "" {
		errorList.AddString("Access Key ID must be specified")
	}
	if c.GetSecretAccessKey() == "" {
		errorList.AddString("Secret Access Key must be specified")
	}
	if c.GetRegion() == "" {
		errorList.AddString("Region must be specified")
	}

	return errorList.ToError()
}
