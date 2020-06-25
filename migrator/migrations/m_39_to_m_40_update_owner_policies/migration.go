package m39to40

import (
	bolt "github.com/etcd-io/bbolt"
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/bolthelpers"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/types"
)

type policyFields struct {
	id          string
	name        string
	description string
	rationale   string
	remediation string
	value       string
}

var (
	policyBucketName = []byte("policies")

	ownerAnnotationPolicy = policyFields{
		id:          "1a498d97-0cc2-45f5-b32e-1f3cca6a3113",
		name:        "Required Annotation: Owner/Team",
		description: "Alert on deployments missing the 'owner' or 'team' annotation",
		rationale:   "The 'owner' or 'team' annotation should always be specified so that the deployment can quickly be associated with a specific user or team.",
		remediation: "Redeploy your service and set the 'owner' or 'team' annotation to yourself or your team respectively per organizational standards.",
		value:       "owner|team=.+",
	}

	ownerLabelPolicy = policyFields{
		id:          "550081a1-ad3a-4eab-a874-8eb68fab2bbd",
		name:        "Required Label: Owner/Team",
		description: "Alert on deployments missing the 'owner' or 'team' label",
		rationale:   "The 'owner' or 'team' label should always be specified so that the deployment can quickly be associated with a specific user or team.",
		remediation: "Redeploy your service and set the 'owner' or 'team' label to yourself or your team respectively per organizational standards.",
		value:       "owner|team=.+",
	}

	policyMap = map[string]*policyFields{
		"1a498d97-0cc2-45f5-b32e-1f3cca6a3113": &ownerAnnotationPolicy,
		"550081a1-ad3a-4eab-a874-8eb68fab2bbd": &ownerLabelPolicy,
	}
)

func updatePolicies(db *bolt.DB) error {
	if exists, err := bolthelpers.BucketExists(db, policyBucketName); err != nil {
		return err
	} else if !exists {
		return nil
	}
	policyBucket := bolthelpers.TopLevelRef(db, policyBucketName)
	if len(policyMap) == 0 {
		return errors.New("Policy data has something wrong in the migration program.")
	}

	return policyBucket.Update(func(b *bolt.Bucket) error {
		for key, val := range policyMap {
			v := b.Get([]byte(key))
			if v == nil {
				continue
			}
			var policy storage.Policy
			if err := proto.Unmarshal(v, &policy); err != nil {
				return err
			}

			if val != nil {
				policy.Name = val.name
				policy.Description = val.description
				policy.Rationale = val.rationale
				policy.Remediation = val.remediation

				for _, section := range policy.GetPolicySections() {
					for _, group := range section.GetPolicyGroups() {
						if group.GetFieldName() == "Required Label" || group.GetFieldName() == "Required Annotation" {
							for _, value := range group.GetValues() {
								if value.GetValue() == "owner=.+" {
									value.Value = val.value
									break
								}
							}
						}
					}
				}
			}

			policyBytes, err := proto.Marshal(&policy)
			if err != nil {
				return err
			}
			if err := b.Put([]byte(key), policyBytes); err != nil {
				return errors.Wrap(err, "failed to insert")
			}
		}
		return nil
	})
}

var (
	migration = types.Migration{
		StartingSeqNum: 39,
		VersionAfter:   storage.Version{SeqNum: 40},
		Run: func(databases *types.Databases) error {
			err := updatePolicies(databases.BoltDB)
			if err != nil {
				return errors.Wrap(err, "updating policy texts")
			}
			return nil
		},
	}
)

func init() {
	migrations.MustRegisterMigration(migration)
}
