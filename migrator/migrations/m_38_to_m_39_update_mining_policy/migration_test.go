package m38tom39

import (
	"testing"

	bolt "github.com/etcd-io/bbolt"
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/bolthelpers"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/suite"
)

func TestMiningPolicyMigration(t *testing.T) {
	suite.Run(t, new(miningPolicyTestSuite))
}

type miningPolicyTestSuite struct {
	suite.Suite

	db *bolt.DB
}

func (suite *miningPolicyTestSuite) SetupTest() {
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

func (suite *miningPolicyTestSuite) TearDownTest() {
	testutils.TearDownDB(suite.db)
}

func insertThing(bucket bolthelpers.BucketRef, id string, pb proto.Message) error {
	return bucket.Update(func(b *bolt.Bucket) error {
		bytes, err := proto.Marshal(pb)
		if err != nil {
			return err
		}
		return b.Put([]byte(id), bytes)
	})
}

func (suite *miningPolicyTestSuite) TestMiningPolicyMigration() {
	cases := []struct {
		oldPolicy      *storage.Policy
		expectedPolicy *storage.Policy
	}{
		{
			oldPolicy: &storage.Policy{
				Id: "bs",
			},
			expectedPolicy: &storage.Policy{
				Id: "bs",
			},
		},
		{
			oldPolicy: &storage.Policy{
				Id: policyID,
				PolicySections: []*storage.PolicySection{
					{
						SectionName: "",
						PolicyGroups: []*storage.PolicyGroup{
							{
								FieldName: processName,
								Values: []*storage.PolicyValue{
									{
										Value: "bs",
									},
								},
							},
						},
					},
				},
			},
			expectedPolicy: &storage.Policy{
				Id: policyID,
				PolicySections: []*storage.PolicySection{
					{
						SectionName: "",
						PolicyGroups: []*storage.PolicyGroup{
							{
								FieldName: processName,
								Values: []*storage.PolicyValue{
									{
										Value: "bs",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			oldPolicy: &storage.Policy{
				Id: policyID,
				PolicySections: []*storage.PolicySection{
					{
						SectionName: "",
						PolicyGroups: []*storage.PolicyGroup{
							{
								FieldName: processName,
								Values: []*storage.PolicyValue{
									{
										Value: oldPolicyCriteria,
									},
								},
							},
						},
					},
				},
			},
			expectedPolicy: &storage.Policy{
				Id: policyID,
				PolicySections: []*storage.PolicySection{
					{
						SectionName: "",
						PolicyGroups: []*storage.PolicyGroup{
							{
								FieldName: processName,
								Values: []*storage.PolicyValue{
									{
										Value: newPolicyCriteria,
									},
								},
							},
						},
					},
				},
			},
		},
	}

	bucket := bolthelpers.TopLevelRef(suite.db, policyBucketName)

	for _, c := range cases {
		suite.NoError(insertThing(bucket, c.oldPolicy.GetId(), c.oldPolicy))
		suite.NoError(updateCryptoMiningPolicyCriteria(suite.db))

		var newPolicy storage.Policy
		suite.NoError(bucket.View(func(b *bolt.Bucket) error {
			v := b.Get([]byte(c.oldPolicy.GetId()))
			return proto.Unmarshal(v, &newPolicy)
		}))
		suite.EqualValues(c.expectedPolicy, &newPolicy)
	}
}
