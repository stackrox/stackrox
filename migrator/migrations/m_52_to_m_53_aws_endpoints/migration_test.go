package m52tom53

import (
	"fmt"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/bolthelpers"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/suite"
	bolt "go.etcd.io/bbolt"
)

func TestAWSEndpointMigration(t *testing.T) {
	suite.Run(t, new(awsEndpointTestSuite))
}

type awsEndpointTestSuite struct {
	suite.Suite

	db *bolt.DB
}

func (suite *awsEndpointTestSuite) SetupTest() {
	db, err := bolthelpers.NewTemp(testutils.DBFileName(suite))
	if err != nil {
		suite.FailNow("Failed to make BoltDB", err.Error())
	}
	suite.NoError(db.Update(func(tx *bolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists(imageIntegrationBucket); err != nil {
			return err
		}
		if _, err := tx.CreateBucketIfNotExists(externalBackupsBucket); err != nil {
			return err
		}
		return nil
	}))
	suite.db = db
}

func (suite *awsEndpointTestSuite) TearDownTest() {
	testutils.TearDownDB(suite.db)
}

func (suite *awsEndpointTestSuite) TestMigrateExternalBackups() {
	externalBackups := []*storage.ExternalBackup{
		{
			Id:   "1",
			Type: "gcs",
			Config: &storage.ExternalBackup_Gcs{
				Gcs: &storage.GCSConfig{
					Bucket:         "bucket",
					ServiceAccount: "serviceaccount",
				},
			},
		},
		{
			Id:   "2",
			Type: "s3",
			Config: &storage.ExternalBackup_S3{
				S3: &storage.S3Config{
					Bucket:   "bucket",
					Endpoint: "random-endpoint",
					Region:   "us-west1",
				},
			},
		},
		{
			Id:   "3",
			Type: "s3",
			Config: &storage.ExternalBackup_S3{
				S3: &storage.S3Config{
					Bucket: "bucket",
					Region: "us-west1",
				},
			},
		},
	}

	err := suite.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(externalBackupsBucket)
		for _, backup := range externalBackups {
			bytes, err := proto.Marshal(backup)
			if err != nil {
				return err
			}
			if err := bucket.Put([]byte(backup.GetId()), bytes); err != nil {
				return err
			}
		}
		return nil
	})
	suite.NoError(err)

	// Migrate the data
	suite.NoError(migrateExternalBackups(suite.db))

	expected := []*storage.ExternalBackup{
		{
			Id:   "1",
			Type: "gcs",
			Config: &storage.ExternalBackup_Gcs{
				Gcs: &storage.GCSConfig{
					Bucket:         "bucket",
					ServiceAccount: "serviceaccount",
				},
			},
		},
		{
			Id:   "2",
			Type: "s3",
			Config: &storage.ExternalBackup_S3{
				S3: &storage.S3Config{
					Bucket:   "bucket",
					Endpoint: "random-endpoint",
					Region:   "us-west1",
				},
			},
		},
		{
			Id:   "3",
			Type: "s3",
			Config: &storage.ExternalBackup_S3{
				S3: &storage.S3Config{
					Bucket:   "bucket",
					Endpoint: "s3.us-west1.amazonaws.com",
					Region:   "us-west1",
				},
			},
		},
	}

	err = suite.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(externalBackupsBucket)
		for _, backup := range expected {
			value := bucket.Get([]byte(backup.GetId()))
			if len(value) == 0 {
				return fmt.Errorf("no value for id: %q", backup.GetId())
			}
			var eb storage.ExternalBackup
			if err := proto.Unmarshal(value, &eb); err != nil {
				return err
			}

			suite.Equal(backup, &eb)
		}
		return nil
	})
	suite.NoError(err)
}

func (suite *awsEndpointTestSuite) TestMigrateImageIntegrations() {
	imageIntegrations := []*storage.ImageIntegration{
		{
			Id:   "1",
			Type: "docker",
			IntegrationConfig: &storage.ImageIntegration_Docker{
				Docker: &storage.DockerConfig{
					Endpoint: "endpoint",
				},
			},
		},
		{
			Id:   "2",
			Type: "ecr",
			IntegrationConfig: &storage.ImageIntegration_Ecr{
				Ecr: &storage.ECRConfig{
					Endpoint: "set-endpoint",
					Region:   "us-west1",
				},
			},
		},
		{
			Id:   "3",
			Type: "ecr",
			IntegrationConfig: &storage.ImageIntegration_Ecr{
				Ecr: &storage.ECRConfig{
					Region: "us-west1",
				},
			},
		},
	}

	err := suite.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(imageIntegrationBucket)
		for _, ii := range imageIntegrations {
			bytes, err := proto.Marshal(ii)
			if err != nil {
				return err
			}
			if err := bucket.Put([]byte(ii.GetId()), bytes); err != nil {
				return err
			}
		}
		return nil
	})
	suite.NoError(err)

	// Migrate the data
	suite.NoError(migrateECR(suite.db))

	expected := []*storage.ImageIntegration{
		{
			Id:   "1",
			Type: "docker",

			IntegrationConfig: &storage.ImageIntegration_Docker{
				Docker: &storage.DockerConfig{
					Endpoint: "endpoint",
				},
			},
		},
		{
			Id:   "2",
			Type: "ecr",
			IntegrationConfig: &storage.ImageIntegration_Ecr{
				Ecr: &storage.ECRConfig{
					Endpoint: "set-endpoint",
					Region:   "us-west1",
				},
			},
		},
		{
			Id:   "3",
			Type: "ecr",
			IntegrationConfig: &storage.ImageIntegration_Ecr{
				Ecr: &storage.ECRConfig{
					Region:   "us-west1",
					Endpoint: "ecr.us-west1.amazonaws.com",
				},
			},
		},
	}

	err = suite.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(imageIntegrationBucket)
		for _, integration := range expected {
			value := bucket.Get([]byte(integration.GetId()))
			if len(value) == 0 {
				return fmt.Errorf("no value for id: %q", integration.GetId())
			}
			var ii storage.ImageIntegration
			if err := proto.Unmarshal(value, &ii); err != nil {
				return err
			}

			suite.Equal(integration, &ii)
		}
		return nil
	})
	suite.NoError(err)
}
