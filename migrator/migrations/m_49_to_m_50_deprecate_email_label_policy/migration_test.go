package m49tom50

import (
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/bolthelpers"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/suite"
	bolt "go.etcd.io/bbolt"
)

func TestDeprecateEmailLabelPolicyMigration(t *testing.T) {
	suite.Run(t, new(emailLabelPolicyTestSuite))
}

type emailLabelPolicyTestSuite struct {
	suite.Suite

	db *bolt.DB
}

func (suite *emailLabelPolicyTestSuite) SetupTest() {
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

func (suite *emailLabelPolicyTestSuite) TearDownTest() {
	testutils.TearDownDB(suite.db)
}

func insertPolicy(bucket bolthelpers.BucketRef, id string, pb proto.Message) error {
	return bucket.Update(func(b *bolt.Bucket) error {
		bytes, err := proto.Marshal(pb)
		if err != nil {
			return err
		}
		return b.Put([]byte(id), bytes)
	})
}

func (suite *emailLabelPolicyTestSuite) TestDeprecateEmailLabelPolicyMigration() {
	bucket := bolthelpers.TopLevelRef(suite.db, policyBucketName)
	policy := &storage.Policy{
		Id: "bs",
	}

	suite.NoError(insertPolicy(bucket, policy.GetId(), policy))
	suite.NoError(deprecateRequiredLabelEmailPolicy(suite.db))

	var newPolicy storage.Policy
	suite.NoError(bucket.View(func(b *bolt.Bucket) error {
		v := b.Get([]byte(policy.GetId()))
		return proto.Unmarshal(v, &newPolicy)
	}))
	suite.EqualValues(policy, &newPolicy)

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
								Value: policyCriteria,
							},
						},
					},
				},
			},
		},
	}

	suite.NoError(insertPolicy(bucket, policy.GetId(), policy))
	suite.NoError(deprecateRequiredLabelEmailPolicy(suite.db))
	suite.NoError(bucket.View(func(b *bolt.Bucket) error {
		v := b.Get([]byte(policy.GetId()))
		return proto.Unmarshal(v, &newPolicy)
	}))
	suite.EqualValues(&storage.Policy{}, &newPolicy)

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
								Value: policyCriteria,
							},
						},
					},
					{
						FieldName: "Required Label: StackRox",
						Values: []*storage.PolicyValue{
							{
								Value: "",
							},
						},
					},
				},
			},
		},
	}

	suite.NoError(insertPolicy(bucket, policy.GetId(), policy))
	suite.NoError(deprecateRequiredLabelEmailPolicy(suite.db))
	suite.NoError(bucket.View(func(b *bolt.Bucket) error {
		v := b.Get([]byte(policy.GetId()))
		return proto.Unmarshal(v, &newPolicy)
	}))
	suite.EqualValues(policy, &newPolicy)
}
