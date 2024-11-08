package gcs

import (
	"context"
	"fmt"
	"io"
	"path"
	"sort"
	"strings"
	"time"

	googleStorage "cloud.google.com/go/storage"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/cloudproviders/gcp"
	"github.com/stackrox/rox/central/externalbackups/plugins"
	"github.com/stackrox/rox/central/externalbackups/plugins/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/administration/events/codes"
	"github.com/stackrox/rox/pkg/administration/events/option"
	gcpUtils "github.com/stackrox/rox/pkg/cloudproviders/gcp/utils"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/logging"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
)

const (
	timeout          = 5 * time.Second
	backupMaxTimeout = 4 * time.Hour

	backupPrefix = "stackrox-backup"
	timeFormat   = "2006-01-02-15-04-05"
)

var log = logging.LoggerForModule(option.EnableAdministrationEvents())

type gcs struct {
	integration *storage.ExternalBackup
	client      *googleStorage.Client

	backupsToKeep int
	bucket        string
	objectPrefix  string
}

func validate(conf *storage.GCSConfig) error {
	errorList := errorhelpers.NewErrorList("GCS Validation")
	if conf.GetBucket() == "" {
		errorList.AddString("Bucket must be specified")
	}
	if conf.GetServiceAccount() == "" && !conf.GetUseWorkloadId() {
		errorList.AddString("Service Account JSON or Use Workload Identity must be specified")
	}
	if conf.GetServiceAccount() != "" && conf.GetUseWorkloadId() {
		errorList.AddString("Service Account JSON must be empty when workload ID is enabled")
	}
	return errorList.ToError()
}

func newGCS(integration *storage.ExternalBackup) (*gcs, error) {
	conf := integration.GetGcs()
	if conf == nil {
		return nil, errors.New("GCS configuration required")
	}
	if err := validate(conf); err != nil {
		return nil, err
	}

	var (
		client *googleStorage.Client
		err    error
	)
	client, err = gcpUtils.CreateStorageClientFromConfigWithManager(context.Background(), conf, gcp.Singleton())
	if err != nil {
		return nil, errors.Wrap(err, "could not create GCS client")
	}
	return &gcs{
		integration:   integration,
		client:        client,
		bucket:        conf.GetBucket(),
		backupsToKeep: int(integration.GetBackupsToKeep()),
		objectPrefix:  conf.GetObjectPrefix(),
	}, nil
}

func (s *gcs) send(client *googleStorage.Client, duration time.Duration, objectPath string, reader io.Reader) error {
	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	bucketHandle := client.Bucket(s.bucket)
	wc := bucketHandle.Object(objectPath).NewWriter(ctx)
	if _, err := io.Copy(wc, reader); err != nil {
		if err := wc.Close(); err != nil {
			log.Errorf("closing GCS writer: %v", err)
		}
		return errors.Wrap(err, "writing backup to GCS stream")
	}
	if err := wc.Close(); err != nil {
		return errors.Wrap(err, "closing GCS writer")
	}
	return nil
}

func (s *gcs) delete(client *googleStorage.Client, objectPath string) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	bucketHandle := client.Bucket(s.bucket)
	err := bucketHandle.Object(objectPath).Delete(ctx)
	if err != nil {
		return errors.Wrapf(err, "deleting object: %s", objectPath)
	}
	return nil
}

func (s *gcs) pruneBackupsIfNecessary(client *googleStorage.Client) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	bucketHandle := client.Bucket(s.bucket)
	objectIterator := bucketHandle.Objects(ctx, &googleStorage.Query{
		Prefix: s.objectPrefix,
	})

	var currentBackups []*googleStorage.ObjectAttrs
	var attrs *googleStorage.ObjectAttrs
	var err error

	trimPrefix := s.prefixKey(backupPrefix)
	for attrs, err = objectIterator.Next(); err == nil; attrs, err = objectIterator.Next() {

		// Defend against other file types in the bucket
		if !strings.HasPrefix(attrs.Name, trimPrefix) {
			continue
		}
		currentBackups = append(currentBackups, attrs)
	}
	if err != iterator.Done {
		return s.createError("fetching all objects from GCS bucket", err)
	}

	if len(currentBackups) <= s.backupsToKeep {
		return nil
	}
	// Sort with earliest created first
	sort.Slice(currentBackups, func(i, j int) bool {
		return currentBackups[i].Created.Before(currentBackups[j].Created)
	})

	errorList := errorhelpers.NewErrorList("remove objects in GCS store")
	numBackupsToRemove := len(currentBackups) - s.backupsToKeep
	for _, attrToRemove := range currentBackups[:numBackupsToRemove] {
		log.Infof("Pruning old backup %s", attrToRemove.Name)
		if err := s.delete(client, attrToRemove.Name); err != nil {
			errorList.AddError(s.createError(
				fmt.Sprintf("deleting object %q from bucket %q", attrToRemove.Name, s.bucket), err),
			)
		}
	}
	return errorList.ToError()
}

func (s *gcs) prefixKey(key string) string {
	return path.Join(s.objectPrefix, key)
}

func formattedTime() string {
	return time.Now().UTC().Format(timeFormat)
}

func (s *gcs) Backup(reader io.ReadCloser) error {
	defer func() {
		if err := reader.Close(); err != nil {
			log.Errorf("Error closing reader: %v", err)
		}
	}()

	key := fmt.Sprintf("%s-%s.zip", backupPrefix, formattedTime())
	formattedKey := s.prefixKey(key)

	log.Infof("Starting GCS Backup for file %v", formattedKey)
	if err := s.send(s.client, backupMaxTimeout, formattedKey, reader); err != nil {
		return s.createError(fmt.Sprintf("creating backup in bucket %q with key %q", s.bucket, formattedKey), err)
	}
	log.Info("Successfully backed up to GCS")
	return s.pruneBackupsIfNecessary(s.client)
}

func (s *gcs) Test() error {
	formattedKey := s.prefixKey(fmt.Sprintf("%s-test-%s", backupPrefix, formattedTime()))
	reader := strings.NewReader("This is a test of the StackRox integration with this bucket")
	if err := s.send(s.client, timeout, formattedKey, reader); err != nil {
		return s.createError(fmt.Sprintf("creating test object %q in bucket %q", formattedKey, s.bucket), err)
	}

	if err := s.delete(s.client, formattedKey); err != nil {
		return s.createError(fmt.Sprintf("deleting test object %q from bucket %q",
			formattedKey, s.bucket), err)
	}
	return nil
}

func (s *gcs) createError(msg string, err error) error {
	if gErr, _ := err.(*googleapi.Error); gErr != nil {
		msg = fmt.Sprintf("GCS backup: %s (code: %d)", msg, gErr.Code)
	} else {
		msg = fmt.Sprintf("GCS backup: %s: %v", msg, err)
	}
	log.Errorw(msg,
		logging.BackupName(s.integration.GetName()),
		logging.Err(err),
		logging.ErrCode(codes.GCSGeneric),
		logging.String("bucket", s.bucket),
		logging.String("object-prefix", s.integration.GetGcs().GetObjectPrefix()),
	)
	return errors.New(msg)
}

func init() {
	plugins.Add(types.GCSType, func(backup *storage.ExternalBackup) (types.ExternalBackup, error) {
		return newGCS(backup)
	})
}
