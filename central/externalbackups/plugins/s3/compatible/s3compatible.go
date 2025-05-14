package s3compatible

import (
	"net/url"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/externalbackups/plugins"
	s3common "github.com/stackrox/rox/central/externalbackups/plugins/s3/common"
	"github.com/stackrox/rox/central/externalbackups/plugins/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/urlfmt"
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
	//
	// Additionally, this new S3 compatible plugin already uses the aws-sdk-v2
	// while the S3 plugin still uses v1.
	// This is to allow backwards compatibility for customers that are using the S3
	// backup integration with GCS buckets. This is not possible to do with v2
	// because GCS alters the Accept-Encoding header, which breaks the v2 request
	// signature. See:
	// https://github.com/aws/aws-sdk-go-v2/issues/1816
	// Tested here:
	// https://github.com/stackrox/stackrox/pull/11761
	// Using the S3 backup integration interoperability with GCS has been deprecated
	// in 4.5

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

func (c *s3compatibleConfigWrapper) GetValidatedEndpoint() (string, error) {
	return validateEndpoint(c.GetEndpoint())
}

func validateEndpoint(endpoint string) (string, error) {
	// The aws-sdk-go-v2 package does not add a default scheme to the endpoint.
	sanitizedEndpoint := urlfmt.FormatURL(endpoint, urlfmt.HTTPS, urlfmt.NoTrailingSlash)
	if _, err := url.Parse(sanitizedEndpoint); err != nil {
		return "", errors.Wrapf(err, "invalid URL %q", endpoint)
	}
	return sanitizedEndpoint, nil
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

func (c *s3compatibleConfigWrapper) GetErrorCode() string {
	return types.S3CompatibleType
}

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
