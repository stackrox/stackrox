package legacy

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/suite"
	bolt "go.etcd.io/bbolt"
)

func TestPolicyStore(t *testing.T) {
	suite.Run(t, new(policyTestSuite))
}

type policyTestSuite struct {
	suite.Suite
	ctx   context.Context
	db    *bolt.DB
	store Store
}

func (s *policyTestSuite) SetupTest() {
	db, err := bolthelper.NewTemp(s.T().Name() + ".db")
	s.NoError(err, "Failed to make BoltDB")
	s.db = db
	s.store = New(db)

	s.ctx = policyCtx
}

func (s *policyTestSuite) TearDownTest() {
	testutils.TearDownDB(s.db)
}

func (s *policyTestSuite) TestRemovedDefaultPolicies() {
	policy := &storage.Policy{
		Id:                 "policy",
		Name:               "policy",
		MitreVectorsLocked: true,
		MitreAttackVectors: []*storage.Policy_MitreAttackVectors{
			{
				Tactic:     "t1",
				Techniques: []string{"tt1", "tt2"},
			},
		},
	}
	s.NoError(s.store.Upsert(s.ctx, policy))

	getAndVerify := func(removed set.StringSet) {
		allPolicies, err := s.store.GetAll(s.ctx)
		s.NoError(err)
		s.Len(allPolicies, removed.Cardinality()+1)
		for _, p := range allPolicies {
			if p.GetId() == policy.GetId() {
				s.Equal(policy, p)
				continue
			}
			s.Contains(removed, p.GetId())
			s.True(p.GetDisabled())
		}
	}

	dps, err := getRawDefaultPolicies()
	s.NoError(err)
	s.Len(dps, 77, "make sure all default policies are loaded successfully")

	// A selected set of policies are removed
	removedSet := set.NewStringSet()
	for _, dp := range dps[:len(dps)/10] {
		s.upsertRemovedDefaultPolicy(dp.GetId())
		removedSet.Add(dp.GetId())
	}
	getAndVerify(removedSet)

	// All policies are removed
	for _, dp := range dps[len(dps)/10:] {
		s.upsertRemovedDefaultPolicy(dp.GetId())
		removedSet.Add(dp.GetId())
	}
	getAndVerify(removedSet)
}

func (s *policyTestSuite) upsertRemovedDefaultPolicy(policyID string) {
	bytes, err := json.Marshal(true)
	s.NoError(err)
	err = s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(removedDefaultPolicyBucket)
		return bucket.Put([]byte(policyID), bytes)
	})
	s.NoError(err)
}
