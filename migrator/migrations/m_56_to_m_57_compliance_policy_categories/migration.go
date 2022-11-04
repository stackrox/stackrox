package m56tom57

import (
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/log"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/types"
	bolt "go.etcd.io/bbolt"
)

const (
	securityBestPractices = "Security Best Practices"
	devOpsBestPractices   = "DevOps Best Practices"
	dockerCIS             = "Docker CIS"
)

type policyUpdate struct {
	newName          string
	removeCategories []string
}

var (
	migration = types.Migration{
		StartingSeqNum: 56,
		VersionAfter:   &storage.Version{SeqNum: 57},
		Run: func(databases *types.Databases) error {
			return migrateNewPolicyCategories(databases.BoltDB)
		},
	}

	policyBucket = []byte("policies")

	// This maps the policy IDs I want to update to the categories I want to remove from that .policy
	policyChanges = map[string]policyUpdate{
		"47cb9e0a-879a-417b-9a8f-de644d7c8a77": {
			newName: "Docker CIS 5.16: Ensure that the host's IPC namespace is not shared",
			removeCategories: []string{
				securityBestPractices,
			},
		},
		"436811e7-892f-4da6-a0f5-8cc459f1b954": {
			newName: "Docker CIS 5.15: Ensure that the host's process namespace is not shared",
			removeCategories: []string{
				securityBestPractices,
			},
		},
		"6abcaa13-9ed6-4109-a1a7-be2e8280e49e": {
			newName: "Docker CIS 5.7: Ensure privileged ports are not mapped within containers",
			removeCategories: []string{
				securityBestPractices,
			},
		},
		"dce17697-1b72-49d2-b18a-05d893cd9368": {
			newName: "Docker CIS 4.1: Ensure That a User for the Container Has Been Created",
			removeCategories: []string{
				securityBestPractices,
				devOpsBestPractices,
			},
		},
		"9a91b4de-d52e-4d4d-a65e-1e785c3501b1": {
			newName: "Docker CIS 4.7: Alert on Update Instruction",
			removeCategories: []string{
				securityBestPractices,
				devOpsBestPractices,
			},
		},
		"32d770b9-c6ba-4398-b48a-0c3e807644ed": {
			newName: "Docker CIS 5.19: Ensure mount propagation mode is not enabled",
			removeCategories: []string{
				securityBestPractices,
			},
		},
		"6226d4ad-7619-4a0b-a160-46373cfcee66": {
			newName: "Docker CIS 5.9 and 5.20: Ensure that the host's network namespace is not shared",
			removeCategories: []string{
				securityBestPractices,
			},
		},
		"41e5153f-98d1-4830-9f80-983afcbe73c1": {
			newName: "Docker CIS 5.21: Ensure the default seccomp profile is not disabled",
			removeCategories: []string{
				securityBestPractices,
			},
		},
	}
)

func init() {
	migrations.MustRegisterMigration(migration)
}

func migrateNewPolicyCategories(db *bolt.DB) error {
	err := db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(policyBucket)
		if bucket == nil {
			return errors.Errorf("bucket %q not found", policyBucket)
		}

		for migrateID, policyChange := range policyChanges {
			policyBytes := bucket.Get([]byte(migrateID))
			if policyBytes == nil {
				log.WriteToStderrf("no policy exists for ID %s in policy category migration.  Continuing", migrateID)
				continue
			}

			policy := &storage.Policy{}
			if err := proto.Unmarshal(policyBytes, policy); err != nil {
				// Unclear how to recover from unmarshal error, abort the transaction.
				return errors.Wrapf(err, "failed to unmarshal policy data for key %q", migrateID)
			}
			migratePolicy(policy, policyChange.removeCategories)
			if policyChange.newName != "" {
				policy.Name = policyChange.newName
			}

			obj, err := proto.Marshal(policy)
			if err != nil {
				// Unclear how to recover from marshal error, abort the transaction.
				return errors.Wrapf(err, "failed to marshal migrated policy %q for key %q", policy.GetName(), policy.GetId())
			}

			// Update successfully migrated policy. No need to update secondary
			// mappings because neither policy name nor id has changed.
			if err := bucket.Put([]byte(migrateID), obj); err != nil {
				// Unclear how to recover if we cannot update the record.
				return errors.Wrapf(err, "failed to write migrated policy with key %q to the store", migrateID)
			}
		}

		return nil
	})

	if err == nil {
		log.WriteToStderrf("successfully migrated Docker CIS policy categories")
	}

	return err
}

func migratePolicy(p *storage.Policy, removeCategories []string) {
	for _, toRemove := range removeCategories {
		removeCategory(p, toRemove)
	}

	p.Categories = append(p.Categories, dockerCIS)
}

func removeCategory(p *storage.Policy, toRemove string) {
	categories := p.GetCategories()
	for i, existingCategory := range categories {
		if toRemove == existingCategory {
			p.Categories = append(categories[:i], categories[i+1:]...)
			return
		}
	}
}
