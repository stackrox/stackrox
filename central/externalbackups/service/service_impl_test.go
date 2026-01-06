package service

import (
	"context"
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestCloudBackupCapabilityValidation(t *testing.T) {
	backup := &storage.ExternalBackup{
		Name:          "S3 Backup",
		BackupsToKeep: 5,
		Schedule:      &storage.Schedule{IntervalType: storage.Schedule_DAILY},
		Config: &storage.ExternalBackup_S3{
			S3: &storage.S3Config{
				Bucket:   "test-bucket",
				Region:   "us-east-1",
				Endpoint: "s3.amazonaws.com",
			},
		},
	}

	t.Run("allow cloud backup when not managed", func(t *testing.T) {
		t.Setenv("ROX_MANAGED_CENTRAL", "false")

		err := validateBackup(backup)
		assert.NoError(t, err)
	})

	t.Run("block cloud backup when managed", func(t *testing.T) {
		t.Setenv("ROX_MANAGED_CENTRAL", "true")

		err := validateBackup(backup)
		assert.Error(t, err)
		assert.ErrorContains(t, err, "cloud backup integrations")
	})
}

func TestTriggerExternalBackupCapabilityCheck(t *testing.T) {
	s := &serviceImpl{}

	t.Run("block trigger when managed", func(t *testing.T) {
		t.Setenv("ROX_MANAGED_CENTRAL", "true")

		_, err := s.TriggerExternalBackup(context.Background(), &v1.ResourceByID{Id: "test-id"})
		assert.Error(t, err)
		assert.ErrorContains(t, err, "cloud backup integrations")
	})
}
