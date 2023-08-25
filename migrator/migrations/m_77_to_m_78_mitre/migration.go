package m77tom78

import (
	"embed"
	"encoding/json"
	"path/filepath"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/log"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/types"
	bolt "go.etcd.io/bbolt"
)

var (
	migration = types.Migration{
		StartingSeqNum: 77,
		VersionAfter:   &storage.Version{SeqNum: 78},
		Run: func(databases *types.Databases) error {
			err := updatePoliciesWithMitre(databases.BoltDB)
			if err != nil {
				return errors.Wrap(err, "updating system policies with MITRE ATT&CK")
			}
			return nil
		},
	}

	policyBucket = []byte("policies")
	//go:embed policies/*.json
	policiesFS embed.FS
)

func init() {
	migrations.MustRegisterMigration(migration)
}

func updatePoliciesWithMitre(db *bolt.DB) error {
	policies, err := defaultPolicies()
	if err != nil {
		return errors.Wrap(err, "could not read default system policies")
	}

	migratedPolicies := 0
	err = db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(policyBucket)
		if bucket == nil {
			return errors.Errorf("bucket %q not found", policyBucket)
		}

		for _, policy := range policies {
			key := []byte(policy.ID)
			val := bucket.Get(key)
			if val == nil {
				log.WriteToStderrf("default system policy with ID %s not found in DB. Continuing", key)
				continue
			}

			storedPolicy := &storage.Policy{}
			if err := proto.Unmarshal(val, storedPolicy); err != nil {
				return errors.Wrapf(err, "unmarshaling policy with id %q", key)
			}

			storedPolicy.MitreAttackVectors = policy.MitreAttackVectors
			storedPolicy.MitreVectorsLocked = policy.MitreVectorsLocked

			data, err := proto.Marshal(storedPolicy)
			if err != nil {
				return errors.Wrapf(err, "marshalling policy %s", key)
			}

			if err := bucket.Put(key, data); err != nil {
				return errors.Wrapf(err, "adding policy %s", key)
			}
			migratedPolicies++
		}
		return nil
	})
	log.WriteToStderrf("Updated %d/%d default system policies with MITRE ATT&CK", migratedPolicies, len(policies))
	return err
}

func defaultPolicies() ([]*slimPolicy, error) {
	files, err := policiesFS.ReadDir("policies")
	if err != nil {
		return nil, errors.Wrap(err, "could not read default system policies json")
	}

	var policies []*slimPolicy
	for _, f := range files {
		p, err := readPolicyFile(filepath.Join("policies", f.Name()))
		if err != nil {
			log.WriteToStderrf(err.Error())
			continue
		}
		if p.ID == "" {
			return nil, errors.Errorf("policy %s does not have an ID defined", p.Name)
		}
		policies = append(policies, p)
	}

	return policies, nil
}

func readPolicyFile(path string) (*slimPolicy, error) {
	contents, err := policiesFS.ReadFile(path)
	if err != nil {
		return nil, errors.Wrapf(err, "could not read default system policy %s", path)
	}

	var policy slimPolicy
	err = json.Unmarshal(contents, &policy)
	if err != nil {
		return nil, errors.Wrapf(err, "Unable to unmarshal policy (%s) json", path)
	}
	return &policy, nil
}

type slimPolicy struct {
	ID                 string                               `json:"id"`
	Name               string                               `json:"name"`
	MitreAttackVectors []*storage.Policy_MitreAttackVectors `json:"mitreAttackVectors"`
	MitreVectorsLocked bool                                 `json:"mitreVectorsLocked"`
}
