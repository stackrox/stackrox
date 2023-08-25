//go:build externalbackups

package tests

import (
	"context"
	"os"
	"testing"
	"time"

	googleStorage "cloud.google.com/go/storage"
	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/retry"
	"github.com/stackrox/rox/pkg/testutils/centralgrpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

const (
	testGCSBucket = "stackrox-ci-gcs-db-upload-test"
)

func countNumBackups(t *testing.T, client *googleStorage.Client, prefix string) int {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	it := client.Bucket(testGCSBucket).Objects(ctx, &googleStorage.Query{
		Prefix: prefix,
	})
	numBackups := 0
	var err error
	for _, err = it.Next(); err == nil; _, err = it.Next() {
		numBackups++
	}
	require.Equal(t, iterator.Done, err)

	return numBackups
}

func verifyNumBackups(t *testing.T, numBackups int, numExpected int) {
	assert.Equal(t, numExpected, numBackups)
}

func TestGCSExternalBackup(t *testing.T) {
	serviceAccount := os.Getenv("GOOGLE_GCS_BACKUP_SERVICE_ACCOUNT")
	require.NotEmpty(t, serviceAccount)

	prefix := os.Getenv("BUILD_ID")
	require.NotEmpty(t, prefix)

	client, err := googleStorage.NewClient(context.Background(), option.WithCredentialsJSON([]byte(serviceAccount)))
	require.NoError(t, err)

	conn := centralgrpc.GRPCConnectionToCentral(t)
	service := v1.NewExternalBackupServiceClient(conn)

	externalBackup := &storage.ExternalBackup{
		Name:          "GCS backup",
		Type:          "gcs",
		BackupsToKeep: 2,
		Schedule: &storage.Schedule{
			IntervalType: storage.Schedule_DAILY,
			Hour:         3,
			Minute:       0,
		},
		Config: &storage.ExternalBackup_Gcs{
			Gcs: &storage.GCSConfig{
				Bucket:         testGCSBucket,
				ServiceAccount: os.Getenv("GOOGLE_GCS_BACKUP_SERVICE_ACCOUNT"),
				ObjectPrefix:   prefix,
			},
		},
	}

	// We could be in a situation where central isn't quite ready from the
	// previous tests.  This will retry a few times until it is if that is the case.
	// If this first one doesn't work, then the rest are doomed so no need to wrap those
	// in retries.
	err = retry.WithRetry(func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		_, err := service.TestExternalBackup(ctx, externalBackup)
		cancel()
		return err
	},
		retry.Tries(10),
		retry.BetweenAttempts(func(_ int) {
			time.Sleep(10 * time.Second)
		}),
		retry.OnFailedAttempts(func(err error) {
			log.Error(err.Error())
		}),
	)
	assert.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	backup, err := service.PostExternalBackup(ctx, externalBackup)
	assert.NoError(t, err)
	cancel()

	verifyNumBackups(t, countNumBackups(t, client, prefix), 0)

	ctx, cancel = context.WithTimeout(context.Background(), 2*time.Minute)
	_, err = service.TriggerExternalBackup(ctx, &v1.ResourceByID{Id: backup.GetId()})
	assert.NoError(t, err)
	cancel()

	verifyNumBackups(t, countNumBackups(t, client, prefix), 1)

	ctx, cancel = context.WithTimeout(context.Background(), 2*time.Minute)
	_, err = service.TriggerExternalBackup(ctx, &v1.ResourceByID{Id: backup.GetId()})
	assert.NoError(t, err)
	cancel()

	verifyNumBackups(t, countNumBackups(t, client, prefix), 2)

	ctx, cancel = context.WithTimeout(context.Background(), 2*time.Minute)
	_, err = service.TriggerExternalBackup(ctx, &v1.ResourceByID{Id: backup.GetId()})
	assert.NoError(t, err)
	cancel()

	// Should have pruned the first one
	err = retry.WithRetry(func() error {
		numBackups := countNumBackups(t, client, prefix)
		if numBackups != 2 {
			return errors.Errorf("Backup is not pruned: got %d", numBackups)
		}
		return nil
	},
		retry.Tries(10),
		retry.BetweenAttempts(func(_ int) {
			time.Sleep(1 * time.Second)
		}),
		retry.OnFailedAttempts(func(err error) {
			log.Error(err.Error())
		}),
	)
	require.NoError(t, err)
}
