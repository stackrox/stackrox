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

func (s *s3) upload(duration time.Duration, key string, body io.Reader) error {
	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	input := &sdkS3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
		Body:   body,
	}
	_, err := s.awsUploader.Upload(ctx, input)
	return err
}

func (s *s3) delete(key string) error {
	deleteInput := &sdkS3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}
	_, err := s.awsClient.DeleteObject(context.Background(), deleteInput)
	return err
}

func sortS3Objects(objects []*s3Types.Object) {
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

func (s *s3) pruneBackupsIfNecessary() error {
	listedBackups, err := s.awsClient.ListObjects(context.Background(), &sdkS3.ListObjectsInput{
		Bucket: aws.String(s.integration.GetS3().GetBucket()),
		Prefix: aws.String(s.prefixKey("backup")),
	})
	if err != nil {
		return s.createError(fmt.Sprintf("listing objects in s3 bucket %q", s.bucket), err)
	}

	backups := make([]*s3Types.Object, 0, len(listedBackups.Contents))
	for _, b := range listedBackups.Contents {
		backups = append(backups, &b)
	}
	sortS3Objects(backups)

	var objectsToRemove []*s3Types.Object
	if len(backups) > int(s.integration.GetBackupsToKeep()) {
		objectsToRemove = backups[s.integration.GetBackupsToKeep():]
	}

	errorList := errorhelpers.NewErrorList("remove objects in s3 store")
	for _, o := range objectsToRemove {
		err = s.delete(*o.Key)
		if err != nil {
			errorList.AddError(
				s.createError(
					fmt.Sprintf("deleting object %q from bucket %q", *o.Key, s.bucket),
					err,
				),
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
	if err := s.upload(backupMaxTimeout, formattedKey, reader); err != nil {
		return s.createError(
			fmt.Sprintf("creating backup in bucket %q with key %q", s.bucket, formattedKey),
			err,
		)
	}
	log.Info("Successfully backed up to S3")
	return s.pruneBackupsIfNecessary()
}

func (s *s3) Test() error {
	formattedKey := s.prefixKey("test")
	testBody := strings.NewReader("This is a test of the StackRox integration with this bucket")
	if err := s.upload(testMaxTimeout, formattedKey, testBody); err != nil {
		return s.createError(
			fmt.Sprintf("error creating test object %q in bucket %q", formattedKey, s.bucket),
			err,
		)
	}

	if err := s.delete(formattedKey); err != nil {
		return s.createError(
			fmt.Sprintf("deleting test object %q from bucket %q", formattedKey, s.bucket),
			err,
		)
	}
	return nil
}

func (s *s3) createError(msg string, err error) error {
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		if apiErr.ErrorMessage() != "" {
			msg = fmt.Sprintf(
				"S3 backup: %s (code: %s; message: %s)",
				msg,
				apiErr.ErrorCode(),
				apiErr.ErrorMessage(),
			)
		} else {
			msg = fmt.Sprintf(
				"S3 backup: %s (code: %s)",
				msg,
				apiErr.ErrorCode(),
			)
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
