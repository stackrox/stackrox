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
	"github.com/stackrox/rox/central/externalbackups/plugins"
	"github.com/stackrox/rox/central/externalbackups/plugins/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/logging"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

const (
	timeout          = 5 * time.Second
	backupMaxTimeout = 4 * time.Hour

	backupPrefix = "stackrox-backup"
	timeFormat   = "2006-01-02-15-04-05"
)

var (
	log = logging.LoggerForModule()
)

type gcs struct {
	integration  *storage.ExternalBackup
	bucketHandle *googleStorage.BucketHandle

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

func getClient(conf *storage.GCSConfig) (*googleStorage.Client, error) {
	if conf.GetUseWorkloadId() {
		return googleStorage.NewClient(context.Background())
	}
	return googleStorage.NewClient(context.Background(), option.WithCredentialsJSON([]byte(conf.GetServiceAccount())))
}

func newGCS(integration *storage.ExternalBackup) (*gcs, error) {
	conf := integration.GetGcs()
	if conf == nil {
		return nil, errors.New("GCS configuration required")
	}
	if err := validate(conf); err != nil {
		return nil, err
	}

	client, err := getClient(conf)
	if err != nil {
		return nil, errors.Wrap(err, "could not create GCS client")
	}
	return &gcs{
		integration:   integration,
		bucketHandle:  client.Bucket(conf.GetBucket()),
		bucket:        conf.GetBucket(),
		backupsToKeep: int(integration.GetBackupsToKeep()),
		objectPrefix:  conf.GetObjectPrefix(),
	}, nil
}

func (s *gcs) send(duration time.Duration, objectPath string, reader io.Reader) error {
	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	wc := s.bucketHandle.Object(objectPath).NewWriter(ctx)
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

func (s *gcs) delete(objectPath string) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	err := s.bucketHandle.Object(objectPath).Delete(ctx)
	if err != nil {
		return errors.Wrapf(err, "deleting object: %s", objectPath)
	}
	return nil
}

func (s *gcs) pruneBackupsIfNecessary() {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	objectIterator := s.bucketHandle.Objects(ctx, &googleStorage.Query{
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
		log.Errorf("fetching all objects from GCS bucket: %v", err)
		return
	}

	if len(currentBackups) <= s.backupsToKeep {
		return
	}
	// Sort with earliest created first
	sort.Slice(currentBackups, func(i, j int) bool {
		return currentBackups[i].Created.Before(currentBackups[j].Created)
	})

	numBackupsToRemove := len(currentBackups) - s.backupsToKeep
	for _, attrToRemove := range currentBackups[:numBackupsToRemove] {
		log.Infof("Pruning old backup %s", attrToRemove.Name)
		if err := s.delete(attrToRemove.Name); err != nil {
			log.Errorf("deleting element %s: %v", attrToRemove.Name, err)
			return
		}
	}
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
	if err := s.send(backupMaxTimeout, formattedKey, reader); err != nil {
		return s.createError(fmt.Sprintf("error creating backup in bucket %q with key %q", s.bucket, formattedKey), err)
	}
	log.Info("Successfully backed up to GCS")
	go s.pruneBackupsIfNecessary()
	return nil
}

func (s *gcs) Test() error {
	formattedKey := s.prefixKey(fmt.Sprintf("%s-test-%s", backupPrefix, formattedTime()))
	reader := strings.NewReader("This is a test of the StackRox integration with this bucket")
	if err := s.send(timeout, formattedKey, reader); err != nil {
		return s.createError(fmt.Sprintf("error creating test object %q in bucket %q", formattedKey, s.bucket), err)
	}

	if err := s.delete(formattedKey); err != nil {
		return s.createError("deleting test object", err)
	}
	return nil
}

func (s *gcs) createError(msg string, err error) error {
	if gErr, _ := err.(*googleapi.Error); gErr != nil {
		msg = fmt.Sprintf("%s (code: %d)", msg, gErr.Code)
	} else {
		msg = fmt.Sprintf("%s: %v", msg, err)
	}
	log.Errorf("GCS backup error: %v", err)
	return errors.New(msg)
}

func init() {
	plugins.Add("gcs", func(backup *storage.ExternalBackup) (types.ExternalBackup, error) {
		return newGCS(backup)
	})
}
