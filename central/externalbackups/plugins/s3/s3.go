package s3

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	credentialsV2 "github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	sdkS3 "github.com/aws/aws-sdk-go-v2/service/s3"
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
)

const (
	backupMaxTimeout = 4 * time.Hour
	// Keep test timeout smaller than the UI timeout (see apps/platform/src/services/instance.js#7).
	testMaxTimeout                 = 9 * time.Second
	initialConfigurationMaxTimeout = 5 * time.Minute
)

var log = logging.LoggerForModule(option.EnableAdministrationEvents())

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
type s3 struct {
	integration *storage.ExternalBackup
	bucket      string
	awsClient   *sdkS3.Client
	awsUploader *manager.Uploader
}

func validate(conf *storage.S3Config) error {
	errorList := errorhelpers.NewErrorList("S3 Validation")
	if conf.GetBucket() == "" {
		errorList.AddString("Bucket must be specified")
	}
	if !conf.GetUseIam() {
		if conf.GetAccessKeyId() == "" {
			errorList.AddString("Access Key ID must be specified")
		}
		if conf.GetSecretAccessKey() == "" {
			errorList.AddString("Secret Access Key must be specified")
		}
	} else if conf.GetAccessKeyId() != "" || conf.GetSecretAccessKey() != "" {
		errorList.AddStrings("IAM and access/secret key use are mutually exclusive. Only specify one")
	}
	if conf.GetRegion() == "" {
		errorList.AddString("Region must be specified")
	}
	return errorList.ToError()
}

func newS3(integration *storage.ExternalBackup) (*s3, error) {
	s3Config, ok := integration.Config.(*storage.ExternalBackup_S3)
	if !ok {
		return nil, errors.New("S3 configuration required")
	}
	conf := s3Config.S3
	if err := validate(conf); err != nil {
		return nil, err
	}

	cfgOptions := []func(options *config.LoadOptions) error{
		config.WithRegion(conf.GetRegion()),
		config.WithHTTPClient(&http.Client{Transport: proxy.RoundTripper()}),
	}
	if !conf.GetUseIam() {
		cfgOptions = append(
			cfgOptions,
			config.WithCredentialsProvider(
				credentialsV2.NewStaticCredentialsProvider(
					conf.GetAccessKeyId(),
					conf.GetSecretAccessKey(),
					"",
				),
			),
		)
	}
	ctx, cancel := context.WithTimeout(context.Background(), initialConfigurationMaxTimeout)
	defer cancel()
	awsCfg, err := config.LoadDefaultConfig(ctx, cfgOptions...)
	if err != nil {
		return nil, err
	}

	var clientOpts []func(*sdkS3.Options)
	endpoint := conf.GetEndpoint()
	if endpoint != "" {
		clientOpts = append(clientOpts, func(options *sdkS3.Options) {
			options.BaseEndpoint = aws.String(endpoint)
		})
	}
	awsClient := sdkS3.NewFromConfig(awsCfg, clientOpts...)

	return &s3{
		integration: integration,
		bucket:      integration.GetS3().GetBucket(),
		awsClient:   awsClient,
		awsUploader: manager.NewUploader(awsClient),
	}, nil
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

func (s *s3) pruneBackupsIfNecessary(ctx context.Context) error {
	listedBackups, err := s.awsClient.ListObjectsV2(context.Background(), &sdkS3.ListObjectsV2Input{
		Bucket: aws.String(s.integration.GetS3().GetBucket()),
		Prefix: aws.String(s.prefixKey("backup")),
	})
	if err != nil {
		return s.createError(fmt.Sprintf("listing objects in s3 bucket %q", s.bucket), err)
	}

	sortS3Objects(listedBackups.Contents)

	var objectsToRemove []s3Types.Object
	if len(listedBackups.Contents) > int(s.integration.GetBackupsToKeep()) {
		objectsToRemove = listedBackups.Contents[s.integration.GetBackupsToKeep():]
	}

	errorList := errorhelpers.NewErrorList("remove objects in s3 store")
	for _, o := range objectsToRemove {
		_, err := s.awsClient.DeleteObject(ctx, &sdkS3.DeleteObjectInput{
			Bucket: aws.String(s.integration.GetS3().GetBucket()),
			Key:    o.Key,
		})
		if err != nil {
			errorList.AddError(s.createError(
				fmt.Sprintf("deleting object %q from bucket %q", *o.Key, s.bucket), err),
			)
		}
	}
	return errorList.ToError()
}

func (s *s3) prefixKey(key string) string {
	return filepath.Join(s.integration.GetS3().GetObjectPrefix(), key)
}

func (s *s3) Backup(reader io.ReadCloser) error {
	defer func() {
		if err := reader.Close(); err != nil {
			log.Errorf("Error closing reader: %v", err)
		}
	}()

	log.Info("Starting S3 Backup")
	formattedTime := time.Now().Format("2006-01-02T15:04:05")
	key := fmt.Sprintf("backup_%s.zip", formattedTime)
	formattedKey := s.prefixKey(key)
	ui := &sdkS3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(formattedKey),
		Body:   reader,
	}
	ctx, cancel := context.WithTimeout(context.Background(), backupMaxTimeout)
	defer cancel()
	if _, err := s.awsUploader.Upload(ctx, ui); err != nil {
		return s.createError(fmt.Sprintf("creating backup in bucket %q with key %q",
			s.bucket, formattedKey), err)
	}
	log.Info("Successfully backed up to S3")
	return s.pruneBackupsIfNecessary(ctx)
}

func (s *s3) Test() error {
	ctx, cancel := context.WithTimeout(context.Background(), testMaxTimeout)
	defer cancel()
	formattedKey := s.prefixKey("test")
	ui := &sdkS3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(formattedKey),
		Body:   strings.NewReader("This is a test of the StackRox integration with this bucket"),
	}
	if _, err := s.awsUploader.Upload(ctx, ui); err != nil {
		return s.createError(fmt.Sprintf("error creating test object %q in bucket %q",
			formattedKey, s.bucket), err)
	}

	if _, err := s.awsClient.DeleteObject(ctx, &sdkS3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(formattedKey),
	}); err != nil {
		return s.createError(fmt.Sprintf("deleting test object %q from bucket %q",
			formattedKey, s.bucket), err)
	}
	return nil
}

func (s *s3) createError(msg string, err error) error {
	var awsErr smithy.APIError
	if errors.As(err, &awsErr) {
		if awsErr.ErrorMessage() != "" {
			msg = fmt.Sprintf("S3 backup: %s (code: %s; message: %s)", msg, awsErr.ErrorCode(), awsErr.ErrorMessage())
		} else {
			msg = fmt.Sprintf("S3 backup: %s (code: %s)", msg, awsErr.ErrorCode())
		}
	}
	log.Errorw(msg,
		logging.BackupName(s.integration.GetName()),
		logging.Err(err),
		logging.ErrCode(codes.S3Generic),
		logging.String("bucket", s.bucket),
		logging.String("object-prefix", s.integration.GetS3().GetObjectPrefix()),
	)
	return errors.New(msg)
}

func init() {
	plugins.Add(types.S3Type, func(backup *storage.ExternalBackup) (types.ExternalBackup, error) {
		return newS3(backup)
	})
}
