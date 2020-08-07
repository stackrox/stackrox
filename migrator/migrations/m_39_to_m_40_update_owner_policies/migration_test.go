package m39to40

import (
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/bolthelpers"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/suite"
	bolt "go.etcd.io/bbolt"
)

func TestMigration(t *testing.T) {
	suite.Run(t, new(MigrationTestSuite))
}

type MigrationTestSuite struct {
	suite.Suite
	db *bolt.DB
}

func (suite *MigrationTestSuite) SetupTest() {
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

func (suite *MigrationTestSuite) TearDownTest() {
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

func (suite *MigrationTestSuite) mustInsertPolicy(p *storage.Policy) {
	policyBucket := bolthelpers.TopLevelRef(suite.db, policyBucketName)
	suite.NoError(insertThing(policyBucket, p.GetId(), p))
}

func (suite *MigrationTestSuite) TestUpdateOwnerPolicies() {
	oldPolicies := []*storage.Policy{
		//Required Annotation: Owner
		{
			Id:   "1a498d97-0cc2-45f5-b32e-1f3cca6a3113",
			Name: "Required Annotation: Owner",

			Description: "Alert on deployments missing the 'owner' annotation",
			Rationale:   "The 'owner' annotation should always be specified so that the deployment can quickly be associated with a specific user or team.",
			Remediation: "Redeploy your service and set the 'owner' annotation to yourself or your team per organizational standards.",

			PolicySections: []*storage.PolicySection{
				{
					SectionName: "",
					PolicyGroups: []*storage.PolicyGroup{
						{
							FieldName:       "Required Annotation",
							BooleanOperator: storage.BooleanOperator_OR,
							Negate:          false,
							Values: []*storage.PolicyValue{
								{
									Value: "owner=.+",
								},
							},
						},
					},
				},
			},
		},
		//Required Label: Owner
		{
			Id:   "550081a1-ad3a-4eab-a874-8eb68fab2bbd",
			Name: "Required Label: Owner",

			Description: "Alert on deployments missing the 'owner' label",
			Rationale:   "The 'owner' label should always be specified so that the deployment can quickly be associated with a specific user or team.",
			Remediation: "Redeploy your service and set the 'owner' label to yourself or your team per organizational standards.",

			PolicySections: []*storage.PolicySection{
				{
					SectionName: "",
					PolicyGroups: []*storage.PolicyGroup{
						{
							FieldName:       "Required Label",
							BooleanOperator: storage.BooleanOperator_OR,
							Negate:          false,
							Values: []*storage.PolicyValue{
								{
									Value: "owner=.+",
								},
							},
						},
					},
				},
			},
		},
	}

	expectedPolicies := []*storage.Policy{
		//Required Annotation: Owner/Team
		{
			Id:   "1a498d97-0cc2-45f5-b32e-1f3cca6a3113",
			Name: "Required Annotation: Owner/Team",

			Description: "Alert on deployments missing the 'owner' or 'team' annotation",
			Rationale:   "The 'owner' or 'team' annotation should always be specified so that the deployment can quickly be associated with a specific user or team.",
			Remediation: "Redeploy your service and set the 'owner' or 'team' annotation to yourself or your team respectively per organizational standards.",

			PolicySections: []*storage.PolicySection{
				{
					SectionName: "",
					PolicyGroups: []*storage.PolicyGroup{
						{
							FieldName:       "Required Annotation",
							BooleanOperator: storage.BooleanOperator_OR,
							Negate:          false,
							Values: []*storage.PolicyValue{
								{
									Value: "owner|team=.+",
								},
							},
						},
					},
				},
			},
		},
		//Required Label: Owner/Team
		{
			Id:   "550081a1-ad3a-4eab-a874-8eb68fab2bbd",
			Name: "Required Label: Owner/Team",

			Description: "Alert on deployments missing the 'owner' or 'team' label",
			Rationale:   "The 'owner' or 'team' label should always be specified so that the deployment can quickly be associated with a specific user or team.",
			Remediation: "Redeploy your service and set the 'owner' or 'team' label to yourself or your team respectively per organizational standards.",

			PolicySections: []*storage.PolicySection{
				{
					SectionName: "",
					PolicyGroups: []*storage.PolicyGroup{
						{
							FieldName:       "Required Label",
							BooleanOperator: storage.BooleanOperator_OR,
							Negate:          false,
							Values: []*storage.PolicyValue{
								{
									Value: "owner|team=.+",
								},
							},
						},
					},
				},
			},
		},
	}

	for _, p := range oldPolicies {
		suite.mustInsertPolicy(p)
	}

	suite.NoError(migration.Run(&types.Databases{BoltDB: suite.db}))

	actualPolicies := make([]*storage.Policy, 0, len(oldPolicies))
	policyBucket := bolthelpers.TopLevelRef(suite.db, policyBucketName)
	suite.NoError(policyBucket.View(func(b *bolt.Bucket) error {
		return b.ForEach(func(_, v []byte) error {
			var policyValue storage.Policy
			err := proto.Unmarshal(v, &policyValue)
			if err != nil {
				return err
			}
			actualPolicies = append(actualPolicies, &policyValue)
			return nil
		})
	}))
	suite.ElementsMatch(actualPolicies, expectedPolicies)
}
