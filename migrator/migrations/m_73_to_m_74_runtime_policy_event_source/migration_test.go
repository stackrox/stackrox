package m73tom74

import (
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	bolt "go.etcd.io/bbolt"
)

var (
	originalPolicies = []*storage.Policy{
		{
			Id:              "0",
			Name:            "policy 0 - build time policy",
			LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_BUILD},
		},
		{
			Id:              "1",
			Name:            "policy 1 - deploy time policy",
			LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_DEPLOY},
		},
		{
			Id:              "2",
			Name:            "policy 2 - runtime policy",
			LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_RUNTIME},
		},
	}

	expectedPolicies = []*storage.Policy{
		{
			Id:              "0",
			Name:            "policy 0 - build time policy",
			LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_BUILD},
			EventSource:     storage.EventSource_NOT_APPLICABLE,
		},
		{
			Id:              "1",
			Name:            "policy 1 - deploy time policy",
			LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_DEPLOY},
			EventSource:     storage.EventSource_NOT_APPLICABLE,
		},
		{
			Id:              "2",
			Name:            "policy 2 - runtime policy",
			LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_RUNTIME},
			EventSource:     storage.EventSource_DEPLOYMENT_EVENT,
		},
	}
)

func TestDefaultEventSourceMigration(t *testing.T) {
	suite.Run(t, new(defaultEventSourceTestSuite))
}

type defaultEventSourceTestSuite struct {
	suite.Suite
	db *bolt.DB
}

func (s *defaultEventSourceTestSuite) SetupSuite() {
	s.db = testutils.DBForT(s.T())
}

func (s *defaultEventSourceTestSuite) TearDownSuite() {
	testutils.TearDownDB(s.db)
}

func (s *defaultEventSourceTestSuite) TestEventSourceMigration() {
	err := s.db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists(policyBucket)
		if err != nil {
			return err
		}

		for _, policy := range originalPolicies {
			bytes, err := proto.Marshal(policy)
			if err != nil {
				return err
			}
			if err := bucket.Put([]byte(policy.GetId()), bytes); err != nil {
				return err
			}
		}
		return nil
	})
	require.NoError(s.T(), err, "Prepare test policy bucket")

	err = migrateDefaultRuntimeEventSource(s.db)
	require.NoError(s.T(), err, "Run migration")

	var migratedPolicies []*storage.Policy
	err = s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(policyBucket)
		if bucket == nil {
			return errors.Errorf("bucket %q not found", policyBucket)
		}
		return bucket.ForEach(func(_, obj []byte) error {
			policy := &storage.Policy{}
			if err := proto.Unmarshal(obj, policy); err != nil {
				return err
			}
			migratedPolicies = append(migratedPolicies, policy)
			return nil
		})
	})
	require.NoError(s.T(), err, "Read migrated policies from the bucket")

	assert.ElementsMatch(s.T(), expectedPolicies, migratedPolicies)
}
