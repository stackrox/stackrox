package s3compatible

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3Types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/externalbackups/plugins"
	"github.com/stackrox/rox/central/externalbackups/plugins/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/administration/events/codes"
	"github.com/stackrox/rox/pkg/administration/events/option"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/httputil/proxy"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/urlfmt"
)

const (
	backupMaxTimeout = 4 * time.Hour
	// Keep test timeout smaller than the UI timeout (see apps/platform/src/services/instance.js#7).
	testMaxTimeout                 = 9 * time.Second
	initialConfigurationMaxTimeout = 5 * time.Minute
	formatKey                      = "backup_2006-01-02T15:04:05.zip"
)

var log = logging.LoggerForModule(option.EnableAdministrationEvents())

func init() {
	plugins.Add(types.S3CompatibleType, func(backup *storage.ExternalBackup) (types.ExternalBackup, error) {
		return newS3Compatible(backup)
	})
}

// s3Compatible plugin for the S3 compatible backup integration.
// As the official AWS S3 is deprecating the path-style bucket addressing but
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
// in 4.5.
type s3Compatible struct {
	integration *storage.ExternalBackup
	bucket      string

	client   *s3.Client
	uploader *manager.Uploader
}

func validate(cfg *storage.S3Compatible) error {
	errorList := errorhelpers.NewErrorList("S3 Compatible Validation")
	if cfg.GetAccessKeyId() == "" {
		errorList.AddString("Access Key ID must be specified")
	}
	if cfg.GetSecretAccessKey() == "" {
		errorList.AddString("Secret Access Key must be specified")
	}
	if cfg.GetRegion() == "" {
		errorList.AddString("Region must be specified")
	}

	return errorList.ToError()
}

func validateEndpoint(endpoint string) (string, error) {
	// The aws-sdk-go-v2 package does not add a default scheme to the endpoint.
	sanitizedEndpoint := urlfmt.FormatURL(endpoint, urlfmt.HTTPS, urlfmt.NoTrailingSlash)
	if _, err := url.Parse(sanitizedEndpoint); err != nil {
		return "", errors.Wrapf(err, "invalid URL %q", endpoint)
	}
	return sanitizedEndpoint, nil
}

func newS3Compatible(integration *storage.ExternalBackup) (*s3Compatible, error) {
	cfg := integration.GetS3Compatible()
	if cfg == nil {
		return nil, errors.New("S3 Compatible configuration required")
	}
	if err := validate(cfg); err != nil {
		return nil, err
	}

	opts := []func(*config.LoadOptions) error{
		config.WithRegion(cfg.GetRegion()),
		config.WithHTTPClient(&http.Client{Transport: proxy.RoundTripper()}),
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				cfg.GetAccessKeyId(),
				cfg.GetSecretAccessKey(), "",
			),
		),
		config.WithRequestChecksumCalculation(aws.RequestChecksumCalculationWhenRequired),
		config.WithResponseChecksumValidation(aws.ResponseChecksumValidationWhenRequired),
	}

	ctx, cancel := context.WithTimeout(context.Background(), initialConfigurationMaxTimeout)
	defer cancel()
	awsConfig, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, errors.Wrap(err, "unable to load the aws config")
	}

	var clientOpts []func(*s3.Options)
	if cfg.GetUrlStyle() == storage.S3URLStyle_S3_URL_STYLE_PATH {
		clientOpts = append(clientOpts, func(o *s3.Options) {
			o.UsePathStyle = true
		})
	}

	if endpoint := cfg.GetEndpoint(); endpoint != "" {
		endpoint, err = validateEndpoint(endpoint)
		if err != nil {
			return nil, err
		}
		clientOpts = append(clientOpts, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(endpoint)
		})
	}

	client := s3.NewFromConfig(awsConfig, clientOpts...)
	return &s3Compatible{
		integration: integration,
		bucket:      integration.GetS3Compatible().GetBucket(),
		client:      client,
		uploader:    manager.NewUploader(client),
	}, nil
}

func (s *s3Compatible) Backup(reader io.ReadCloser) error {
	defer func() {
		if err := reader.Close(); err != nil {
			log.Errorf("closing reader: %+v", err)
		}
	}()

	log.Info("Starting S3 Compatible Backup")
	ctx, cancel := context.WithTimeout(context.Background(), backupMaxTimeout)
	defer cancel()
	key := time.Now().Format(formatKey)
	formattedKey := s.prefixKey(key)
	if _, err := s.uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(formattedKey),
		Body:   reader,
	}); err != nil {
		return s.createError(fmt.Sprintf("creating backup in bucket %q with key %q",
			s.bucket, formattedKey), err)
	}
	log.Info("Successfully backed up to S3 compatible store")
	return s.pruneBackupsIfNecessary(ctx)
}

func (s *s3Compatible) createError(msg string, err error) error {
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		if apiErr.ErrorMessage() != "" {
			msg = fmt.Sprintf("S3 compatible backup: %s (code: %s; message: %s)",
				msg, apiErr.ErrorCode(), apiErr.ErrorMessage())
		} else {
			msg = fmt.Sprintf("S3 compatible backup: %s (code: %s)", msg, apiErr.ErrorCode())
		}
	}
	log.Errorw(msg,
		logging.BackupName(s.integration.GetName()),
		logging.Err(err),
		logging.ErrCode(codes.S3CompatibleGeneric),
		logging.String("bucket", s.bucket),
		logging.String("object-prefix", s.integration.GetS3Compatible().GetObjectPrefix()),
	)
	return errors.New(msg)
}

func (s *s3Compatible) prefixKey(key string) string {
	return filepath.Join(s.integration.GetS3Compatible().GetObjectPrefix(), key)
}

func sortS3Objects(objects []s3Types.Object) {
	sort.SliceStable(objects, func(i, j int) bool {
		o1, o2 := objects[i], objects[j]
		if o2.LastModified == nil {
			return true
		}
		if o1.LastModified == nil {
			return false
		}

		return o1.LastModified.After(*o2.LastModified)
	})
}

func (s *s3Compatible) pruneBackupsIfNecessary(ctx context.Context) error {
	objects, err := s.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucket),
		Prefix: aws.String(s.prefixKey("backup")),
	})
	if err != nil {
		return s.createError(fmt.Sprintf("listing objects in s3 compatible bucket %q", s.bucket), err)
	}
	// If the number of objects in the bucket is smaller than the configured
	// number of backups to keep, we exit here.
	if len(objects.Contents) <= int(s.integration.GetBackupsToKeep()) {
		return nil
	}

	sortS3Objects(objects.Contents)

	errorList := errorhelpers.NewErrorList("remove objects in S3 compatible store")
	for _, objToRemove := range objects.Contents[s.integration.GetBackupsToKeep():] {
		_, err = s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
			Bucket: aws.String(s.bucket),
			Key:    objToRemove.Key,
		})
		if err != nil {
			errorList.AddError(s.createError(
				fmt.Sprintf("deleting object %q from bucket %q", *objToRemove.Key, s.bucket), err),
			)
		}
	}
	return errorList.ToError()
}

func (s *s3Compatible) Test() error {
	ctx, cancel := context.WithTimeout(context.Background(), testMaxTimeout)
	defer cancel()
	formattedKey := s.prefixKey("test")
	if _, err := s.uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(formattedKey),
		Body:   strings.NewReader("This is a test of the StackRox integration with this bucket"),
	}); err != nil {
		return s.createError(fmt.Sprintf("creating test object %q in bucket %q",
			formattedKey, s.bucket), err)
	}
	if _, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(formattedKey),
	}); err != nil {
		return s.createError(fmt.Sprintf("deleting test object %q from bucket %q",
			formattedKey, s.bucket), err)
	}
	return nil
}
