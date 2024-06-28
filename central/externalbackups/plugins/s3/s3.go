package s3

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
	awsS3 "github.com/aws/aws-sdk-go-v2/service/s3"
	s3Types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
	// smithyHttp "github.com/aws/smithy-go/transport/http"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/externalbackups/plugins"
	"github.com/stackrox/rox/central/externalbackups/plugins/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/httputil/proxy"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/urlfmt"
)

const (
	backupMaxTimeout = 4 * time.Hour
	testMaxTimeout   = 5 * time.Second
)

var log = logging.LoggerForModule()

type s3 struct {
	integration *storage.ExternalBackup
	client      *awsS3.Client
	uploader    *manager.Uploader
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
	backupConfig, ok := integration.Config.(*storage.ExternalBackup_S3)
	if !ok {
		return nil, errors.New("S3 configuration required")
	}
	if err := validate(backupConfig.S3); err != nil {
		return nil, err
	}

	awsOpts := []func(*config.LoadOptions) error{
		config.WithRegion(backupConfig.S3.GetRegion()),
		config.WithHTTPClient(&http.Client{Transport: proxy.RoundTripper()}),
		config.WithClientLogMode(aws.LogSigning | aws.LogRequest),
	}
	if !backupConfig.S3.GetUseIam() {
		awsOpts = append(awsOpts,
			config.WithCredentialsProvider(
				credentials.NewStaticCredentialsProvider(
					backupConfig.S3.GetAccessKeyId(), backupConfig.S3.GetSecretAccessKey(), "",
				),
			),
		)
	}
	awsConfig, err := config.LoadDefaultConfig(context.Background(), awsOpts...)
	if err != nil {
		return nil, err
	}
	endpoint := backupConfig.S3.GetEndpoint()
	if endpoint != "" {
		s3URL := fmt.Sprintf("https://%s", urlfmt.TrimHTTPPrefixes(endpoint))
		if _, err := url.Parse(s3URL); err != nil {
			return nil, errox.InvalidArgs.CausedByf("invalid URL %q", endpoint)
		}
		awsConfig.BaseEndpoint = aws.String(s3URL)
	}
	client := awsS3.NewFromConfig(awsConfig, func(o *awsS3.Options) {
		// Google Cloud Storage alters the Accept-Encoding header, which breaks the v2 request signature.
		// See https://github.com/aws/aws-sdk-go-v2/issues/1816.
		if strings.Contains(endpoint, "storage.googleapis.com") {
			ignoreSigningHeaders(o, []string{"Accept-Encoding"})
		}
	})
	return &s3{
		integration: integration,
		client:      client,
		uploader:    manager.NewUploader(client),
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
	objects, err := s.client.ListObjects(ctx, &awsS3.ListObjectsInput{
		Bucket: aws.String(s.integration.GetS3().GetBucket()),
		Prefix: aws.String(s.prefixKey("backup")),
	})
	if err != nil {
		return s.createError("failed to list objects for s3 bucket", err)
	}
	sortS3Objects(objects.Contents)

	var objectsToRemove []s3Types.Object
	if len(objects.Contents) > int(s.integration.GetBackupsToKeep()) {
		objectsToRemove = objects.Contents[s.integration.GetBackupsToKeep():]
	}

	for _, obj := range objectsToRemove {
		_, err := s.client.DeleteObject(ctx, &awsS3.DeleteObjectInput{
			Bucket: aws.String(s.integration.GetS3().GetBucket()),
			Key:    obj.Key,
		})
		if err != nil {
			return s.createError(
				fmt.Sprintf("failed to remove backup %q from bucket %q", *obj.Key, s.integration.GetS3().GetBucket()),
				err,
			)
		}
	}
	return nil
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
	ctx, cancel := context.WithTimeout(context.Background(), backupMaxTimeout)
	defer cancel()

	formattedTime := time.Now().Format("2006-01-02T15:04:05")
	key := fmt.Sprintf("backup_%s.zip", formattedTime)
	formattedKey := s.prefixKey(key)
	if _, err := s.uploader.Upload(ctx, &awsS3.PutObjectInput{
		Bucket: aws.String(s.integration.GetS3().GetBucket()),
		Key:    aws.String(formattedKey),
		Body:   reader,
	}); err != nil {
		return s.createError(fmt.Sprintf("error creating backup in bucket %q with key %q",
			s.integration.GetS3().GetBucket(), formattedKey), err)
	}
	log.Info("Successfully backed up to S3")
	return s.pruneBackupsIfNecessary(ctx)
}

func (s *s3) Test() error {
	ctx, cancel := context.WithTimeout(context.Background(), testMaxTimeout)
	defer cancel()

	formattedKey := s.prefixKey("test")
	if _, err := s.uploader.Upload(ctx, &awsS3.PutObjectInput{
		Bucket: aws.String(s.integration.GetS3().GetBucket()),
		Key:    aws.String(formattedKey),
		Body:   strings.NewReader("This is a test of the StackRox integration with this bucket"),
	}); err != nil {
		return s.createError(fmt.Sprintf("error creating test object %q in bucket %q",
			formattedKey, s.integration.GetS3().GetBucket()), err)
	}
	_, err := s.client.DeleteObject(ctx, &awsS3.DeleteObjectInput{
		Bucket: aws.String(s.integration.GetS3().GetBucket()),
		Key:    aws.String(formattedKey),
	})
	if err != nil {
		return s.createError(fmt.Sprintf("failed to remove test object %q from bucket %q",
			formattedKey, s.integration.GetS3().GetBucket()), err)
	}
	return nil
}

func (s *s3) createError(msg string, err error) error {
	if awsErr, _ := err.(smithy.APIError); awsErr != nil {
		msg = fmt.Sprintf(
			"%s (code: %s; message: %s; fault: %s)",
			msg, awsErr.ErrorCode(), awsErr.ErrorMessage(), awsErr.ErrorFault(),
		)
	}
	log.Errorf("S3 backup error: %v", err)
	return errors.New(msg)
}

func init() {
	plugins.Add(types.S3Type, func(backup *storage.ExternalBackup) (types.ExternalBackup, error) {
		return newS3(backup)
	})
}
