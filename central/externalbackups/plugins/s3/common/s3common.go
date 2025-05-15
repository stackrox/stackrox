package s3common

import (
	"context"
	"fmt"
	"github.com/stackrox/rox/pkg/urlfmt"
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
	"github.com/stackrox/rox/central/externalbackups/plugins/types"
	"github.com/stackrox/rox/generated/storage"
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

	timeFormat = "2006-01-02T15:04:05"
)

var log = logging.LoggerForModule(option.EnableAdministrationEvents())

// s3Common holds the data for a s3 or compatible backup integration
type s3Common struct {
	config ConfigWrapper
	bucket string

	now func() time.Time

	clientWrapper s3Wrapper
}

// ConfigWrapper is an interface to extract relevant configuration parameters
// from a storage.ExternalBackup object to later instantiate a S3Common backup
// integration instance.
type ConfigWrapper interface {
	GetUrlStyle() storage.S3URLStyle
	GetEndpoint() string
	GetRegion() string
	GetBucket() string
	GetObjectPrefix() string

	GetUseIam() bool
	GetAccessKeyId() string
	GetSecretAccessKey() string

	GetName() string
	GetPluginType() string
	GetBackupsToKeep() int32

	Validate() error
}

// NewS3Client creates an external backup plugin based on the provided
// S3 or compatible configuration.
func NewS3Client(cfg ConfigWrapper) (types.ExternalBackup, error) {
	err := cfg.Validate()
	if err != nil {
		return nil, err
	}

	cfgOptions := []func(o *config.LoadOptions) error{
		config.WithRegion(cfg.GetRegion()),
		config.WithHTTPClient(&http.Client{Transport: proxy.RoundTripper()}),
	}

	if !cfg.GetUseIam() {
		cfgOptions = append(
			cfgOptions,
			config.WithCredentialsProvider(
				credentials.NewStaticCredentialsProvider(
					cfg.GetAccessKeyId(),
					cfg.GetSecretAccessKey(),
					"",
				),
			),
		)
	}

	if !isBaseS3Config(cfg) {
		cfgOptions = append(
			cfgOptions,
			config.WithRequestChecksumCalculation(aws.RequestChecksumCalculationWhenRequired),
			config.WithResponseChecksumValidation(aws.ResponseChecksumValidationWhenRequired),
		)
	}

	ctx, cancel := context.WithTimeout(context.Background(), initialConfigurationMaxTimeout)
	defer cancel()
	awsCfg, err := config.LoadDefaultConfig(ctx, cfgOptions...)
	if err != nil {
		return nil, err
	}

	var clientOpts []func(*s3.Options)
	if cfg.GetUrlStyle() == storage.S3URLStyle_S3_URL_STYLE_PATH {
		clientOpts = append(clientOpts, func(o *s3.Options) {
			o.UsePathStyle = true
		})
	}
	endpoint := cfg.GetEndpoint()
	if endpoint != "" {
		validatedEndpoint, validationErr := validateEndpoint(endpoint)
		if validationErr != nil {
			return nil, validationErr
		}
		clientOpts = append(clientOpts, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(validatedEndpoint)
		})
	}

	awsClient := s3.NewFromConfig(awsCfg, clientOpts...)

	return &s3Common{
		config: cfg,
		bucket: cfg.GetBucket(),

		now: time.Now,

		clientWrapper: &s3WrapperImpl{
			bucket:   cfg.GetBucket(),
			client:   awsClient,
			uploader: manager.NewUploader(awsClient),
		},
	}, nil
}

func validateEndpoint(endpoint string) (string, error) {
	// The aws-sdk-go-v2 package does not add a default scheme to the endpoint.
	sanitizedEndpoint := urlfmt.FormatURL(endpoint, urlfmt.HTTPS, urlfmt.NoTrailingSlash)
	if _, err := url.Parse(sanitizedEndpoint); err != nil {
		return "", errors.Wrapf(err, "invalid URL %q", endpoint)
	}
	return sanitizedEndpoint, nil
}

func (s *s3Common) Backup(reader io.ReadCloser) error {
	defer func() {
		if err := reader.Close(); err != nil {
			log.Errorf("closing reader: %+v", err)
		}
	}()

	log.Infof("Starting %s backup", s.getLogPrefix())
	ctx, cancel := context.WithTimeout(context.Background(), backupMaxTimeout)
	defer cancel()
	formattedTime := s.now().Format(timeFormat)
	key := fmt.Sprintf("backup_%s.zip", formattedTime)
	formattedKey := s.prefixKey(key)
	if err := s.clientWrapper.Upload(ctx, formattedKey, reader); err != nil {
		return s.createError(fmt.Sprintf("creating backup in bucket %q with key %q",
			s.bucket, formattedKey), err)
	}
	log.Infof("Successfully backed up to %s store", s.getLogPrefix())
	return s.pruneBackupsIfNecessary(ctx)
}

func (s *s3Common) Test() error {
	ctx, cancel := context.WithTimeout(context.Background(), testMaxTimeout)
	defer cancel()
	formattedKey := s.prefixKey("test")
	testBody := strings.NewReader("This is a test of the StackRox integration with this bucket")
	if err := s.clientWrapper.Upload(ctx, formattedKey, testBody); err != nil {
		return s.createError(fmt.Sprintf("creating test object %q in bucket %q",
			formattedKey, s.bucket), err)
	}
	if err := s.clientWrapper.Delete(ctx, formattedKey); err != nil {
		return s.createError(fmt.Sprintf("deleting test object %q from bucket %q",
			formattedKey, s.bucket), err)
	}
	return nil
}

func (s *s3Common) prefixKey(key string) string {
	return filepath.Join(s.config.GetObjectPrefix(), key)
}

func isBaseS3Config(cfg ConfigWrapper) bool {
	return cfg.GetPluginType() == types.S3Type
}

func (s *s3Common) getLogPrefix() string {
	if isBaseS3Config(s.config) {
		return "S3"
	}
	return "S3 compatible"
}

func (s *s3Common) createError(msg string, err error) error {
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		if apiErr.ErrorMessage() != "" {
			msg = fmt.Sprintf("%s backup: %s (code: %s; message: %s)",
				s.getLogPrefix(), msg, apiErr.ErrorCode(), apiErr.ErrorMessage())
		} else {
			msg = fmt.Sprintf("%s backup: %s (code: %s)",
				s.getLogPrefix(), msg, apiErr.ErrorCode())
		}
	}
	log.Errorw(msg,
		logging.BackupName(s.config.GetName()),
		logging.Err(err),
		logging.ErrCode(s.config.GetPluginType()),
		logging.String("bucket", s.bucket),
		logging.String("object-prefix", s.config.GetObjectPrefix()),
	)
	return errors.New(msg)
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

func (s *s3Common) pruneBackupsIfNecessary(ctx context.Context) error {
	objects, err := s.clientWrapper.ListObjects(ctx, s.prefixKey("backup"))
	if err != nil {
		return s.createError(fmt.Sprintf("listing objects in %s bucket %q", s.getLogPrefix(), s.bucket), err)
	}
	// If the number of objects in the bucket is smaller than the configured
	// number of backups to keep, we exit here.
	if len(objects.Contents) <= int(s.config.GetBackupsToKeep()) {
		return nil
	}

	sortS3Objects(objects.Contents)

	errorList := errorhelpers.NewErrorList(fmt.Sprintf("remove objects in %s store", s.getLogPrefix()))
	for _, objToRemove := range objects.Contents[s.config.GetBackupsToKeep():] {
		err = s.clientWrapper.Delete(ctx, *objToRemove.Key)
		if err != nil {
			errorList.AddError(s.createError(
				fmt.Sprintf("deleting object %q from bucket %q", *objToRemove.Key, s.bucket), err),
			)
		}
	}
	return errorList.ToError()
}

// s3Wrapper defines an interface for interactions with the s3 bucket
//
//go:generate mockgen-wrapper
type s3Wrapper interface {
	ListObjects(ctx context.Context, keyPrefix string) (*s3.ListObjectsV2Output, error)
	Delete(ctx context.Context, key string) error
	Upload(ctx context.Context, key string, data io.Reader) error
}

type s3WrapperImpl struct {
	bucket   string
	client   *s3.Client
	uploader *manager.Uploader
}

func (w *s3WrapperImpl) ListObjects(ctx context.Context, keyPrefix string) (*s3.ListObjectsV2Output, error) {
	return w.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String(w.bucket),
		Prefix: aws.String(keyPrefix),
	})
}

func (w *s3WrapperImpl) Delete(ctx context.Context, key string) error {
	_, err := w.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(w.bucket),
		Key:    aws.String(key),
	})
	return err
}

func (w *s3WrapperImpl) Upload(ctx context.Context, key string, data io.Reader) error {
	_, err := w.uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket: aws.String(w.bucket),
		Key:    aws.String(key),
		Body:   data,
	})
	return err
}
