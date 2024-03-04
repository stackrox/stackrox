package m60tom61

import (
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/bolthelpers"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/suite"
	bolt "go.etcd.io/bbolt"
)

func TestUpdateNetworkManagementExecutionPolicyMigration(t *testing.T) {
	suite.Run(t, new(networkManagementExecutionPolicyTestSuite))
}

type networkManagementExecutionPolicyTestSuite struct {
	suite.Suite

	db *bolt.DB
}

func (suite *networkManagementExecutionPolicyTestSuite) SetupTest() {
	db, err := bolthelpers.NewTemp(testutils.DBFileName(suite))
	if err != nil {
		suite.FailNow("Failed to make BoltDB", err.Error())
	}
	suite.NoError(db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(policyBucketName)
		return err
	}))
	suite.db = db
}

func (suite *networkManagementExecutionPolicyTestSuite) TearDownTest() {
	testutils.TearDownDB(suite.db)
}

func insertPolicy(bucket bolthelpers.BucketRef, id string, pb protocompat.Message) error {
	return bucket.Update(func(b *bolt.Bucket) error {
		bytes, err := proto.Marshal(pb)
		if err != nil {
			return err
		}
		return b.Put([]byte(id), bytes)
	})
}

func (suite *networkManagementExecutionPolicyTestSuite) TestUpdateNetworkManagementExecutionPolicyMigration() {
	bucket := bolthelpers.TopLevelRef(suite.db, policyBucketName)

	// Test that an unrelated policy isn't updated
	policy := &storage.Policy{
		Id: "this-is-a-random-id-that-should-not-exist",
	}

	suite.NoError(insertPolicy(bucket, policy.GetId(), policy))
	suite.NoError(updateNetworkManagementExecutionPolicy(suite.db))

	var newPolicy storage.Policy
	suite.NoError(bucket.View(func(b *bolt.Bucket) error {
		v := b.Get([]byte(policy.GetId()))
		return proto.Unmarshal(v, &newPolicy)
	}))
	suite.EqualValues(policy, &newPolicy)

	// Test that a policy that matches id, field name and old criteria gets updated to new
	policy = &storage.Policy{
		Id: policyID,
		PolicySections: []*storage.PolicySection{
			{
				SectionName: "",
				PolicyGroups: []*storage.PolicyGroup{
					{
						FieldName: policyFieldName,
						Values: []*storage.PolicyValue{
							{
								Value: oldPolicyCriteria,
							},
						},
					},
				},
			},
		},
	}

	suite.NoError(insertPolicy(bucket, policy.GetId(), policy))
	suite.NoError(updateNetworkManagementExecutionPolicy(suite.db))
	suite.NoError(bucket.View(func(b *bolt.Bucket) error {
		v := b.Get([]byte(policy.GetId()))
		return proto.Unmarshal(v, &newPolicy)
	}))
	policy.PolicySections[0].PolicyGroups[0].Values[0].Value = newPolicyCriteria
	policy.Exclusions = append(policy.Exclusions, newExclusions...)
	suite.EqualValues(policy, &newPolicy)

	// Test that a policy that matches id, field name _but not_ criteria is not updated
	policy = &storage.Policy{
		Id: policyID,
		PolicySections: []*storage.PolicySection{
			{
				SectionName: "",
				PolicyGroups: []*storage.PolicyGroup{
					{
						FieldName: policyFieldName,
						Values: []*storage.PolicyValue{
							{
								Value: "ethtool",
							},
						},
					},
				},
			},
		},
	}

	suite.NoError(insertPolicy(bucket, policy.GetId(), policy))
	suite.NoError(updateNetworkManagementExecutionPolicy(suite.db))
	suite.NoError(bucket.View(func(b *bolt.Bucket) error {
		v := b.Get([]byte(policy.GetId()))
		return proto.Unmarshal(v, &newPolicy)
	}))
	suite.EqualValues(policy, &newPolicy)

	// Test that a policy that matches id, but has multiple policy groups is not updated
	policy = &storage.Policy{
		Id: policyID,
		PolicySections: []*storage.PolicySection{
			{
				SectionName: "",
				PolicyGroups: []*storage.PolicyGroup{
					{
						FieldName: policyFieldName,
						Values: []*storage.PolicyValue{
							{
								Value: oldPolicyCriteria,
							},
						},
					},
					{
						FieldName: "Image OS",
						Values: []*storage.PolicyValue{
							{
								Value: "ubuntu:19.04",
							},
						},
					},
				},
			},
		},
	}

	suite.NoError(insertPolicy(bucket, policy.GetId(), policy))
	suite.NoError(updateNetworkManagementExecutionPolicy(suite.db))
	suite.NoError(bucket.View(func(b *bolt.Bucket) error {
		v := b.Get([]byte(policy.GetId()))
		return proto.Unmarshal(v, &newPolicy)
	}))
	suite.EqualValues(policy, &newPolicy)

	// Test that a policy that matches id, but has multiple policy sections is not updated
	policy = &storage.Policy{
		Id: policyID,
		PolicySections: []*storage.PolicySection{
			{
				SectionName: "",
				PolicyGroups: []*storage.PolicyGroup{
					{
						FieldName: policyFieldName,
						Values: []*storage.PolicyValue{
							{
								Value: oldPolicyCriteria,
							},
						},
					},
				},
			},
			{
				SectionName: "section 2",
				PolicyGroups: []*storage.PolicyGroup{
					{
						FieldName: "Image OS",
						Values: []*storage.PolicyValue{
							{
								Value: "ubuntu:19.04",
							},
						},
					},
				},
			},
		},
	}

	suite.NoError(insertPolicy(bucket, policy.GetId(), policy))
	suite.NoError(updateNetworkManagementExecutionPolicy(suite.db))
	suite.NoError(bucket.View(func(b *bolt.Bucket) error {
		v := b.Get([]byte(policy.GetId()))
		return proto.Unmarshal(v, &newPolicy)
	}))
	suite.EqualValues(policy, &newPolicy)

	// Test that a policy that matches id and value but already has exclusions is not updated
	policy = &storage.Policy{
		Id: policyID,
		PolicySections: []*storage.PolicySection{
			{
				SectionName: "",
				PolicyGroups: []*storage.PolicyGroup{
					{
						FieldName: policyFieldName,
						Values: []*storage.PolicyValue{
							{
								Value: oldPolicyCriteria,
							},
						},
					},
				},
			},
		},
		Exclusions: []*storage.Exclusion{
			{
				Name: "Blah",
				Deployment: &storage.Exclusion_Deployment{
					Name: "blah",
				},
			},
		},
	}

	suite.NoError(insertPolicy(bucket, policy.GetId(), policy))
	suite.NoError(updateNetworkManagementExecutionPolicy(suite.db))
	suite.NoError(bucket.View(func(b *bolt.Bucket) error {
		v := b.Get([]byte(policy.GetId()))
		return proto.Unmarshal(v, &newPolicy)
	}))
	suite.EqualValues(policy, &newPolicy)
}
