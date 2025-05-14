package s3

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/externalbackups/plugins"
	s3common "github.com/stackrox/rox/central/externalbackups/plugins/s3/common"
	"github.com/stackrox/rox/central/externalbackups/plugins/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/urlfmt"
	"net/url"
)

// s3 plugin for the AWS S3 backup integration.
// As the official AWS S3 is deprecating the path-style bucket addressing but
// this style is still used in non-AWS S3 compatible providers, we decided to
// implement a new plugin for the latter.
// Having the two plugins allows for a clear separation between official AWS
// and non-AWS features.
// Having the S3 compatible plugin separate will also give us more freedom to
// change to a different package if the aws-sdk decides to drop the path-style
// option in the future.
//
// The s3 plugin used to use the aws-go-sdk v1 to allow backwards compatibility
// for customers who were using the s3 backup integration with GCS buckets.
// This is not possible anymore now that the s3 plugins uses the aws-go-sdk-v2
// because GCS alters the Accept-Encoding header, which breaks the v2 request
// signature. See:
// https://github.com/aws/aws-sdk-go-v2/issues/1816
// Tested here:
// https://github.com/stackrox/stackrox/pull/11761
// Using the S3 backup integration interoperability with GCS has been deprecated
// in 4.5.

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

func (c *s3configWrapper) GetPluginType() string { return types.S3Type }

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
