package m74tom75

import (
	"strings"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/protobuf/jsonpb"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/types"
	bolt "go.etcd.io/bbolt"
)

var (
	migration = types.Migration{
		StartingSeqNum: 74,
		VersionAfter:   &storage.Version{SeqNum: 75},
		Run: func(databases *types.Databases) error {
			err := migrateSeverityPolicy(databases.BoltDB)
			if err != nil {
				return errors.Wrap(err, "migrating severity policy")
			}
			return nil
		},
	}

	policyBucket = []byte("policies")
)

func init() {
	migrations.MustRegisterMigration(migration)
}

func migrateSeverityPolicy(db *bolt.DB) error {
	var policy storage.Policy
	if err := jsonpb.Unmarshal(strings.NewReader(policyJSON), &policy); err != nil {
		return errors.Wrap(err, "unmarshalling severity policy JSON")
	}

	// Migrations only run on existing installations and therefore if the policy does not exist,
	// we should add it as disabled to not disrupt customer workflows
	return db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(policyBucket)
		if bucket == nil {
			return errors.Errorf("bucket %q not found", policyBucket)
		}

		key := []byte(policy.GetId())
		if bucket.Get(key) != nil {
			return nil
		}

		data, err := proto.Marshal(&policy)
		if err != nil {
			return errors.Wrap(err, "marshalling policy")
		}

		if err := bucket.Put(key, data); err != nil {
			return errors.Wrap(err, "adding severity policy")
		}
		return nil
	})
}

// The same as in the file except for disabled=true
const policyJSON = `{
  "id": "a919ccaf-6b43-4160-ac5d-a405e1440a41",
  "name": "Fixable Severity at least Important",
  "description": "Alert on deployments with fixable vulnerabilities with a Severity Rating at least Important",
  "rationale": "Known vulnerabilities make it easier for adversaries to exploit your application. You can fix these high-severity vulnerabilities by updating to a newer version of the affected component(s).",
  "remediation": "Use your package manager to update to a fixed version in future builds or speak with your security team to mitigate the vulnerabilities.",
  "disabled": true,
  "categories": [
    "Vulnerability Management"
  ],
  "lifecycleStages": [
    "BUILD",
    "DEPLOY"
  ],
  "severity": "HIGH_SEVERITY",
  "enforcementActions": [
    "FAIL_BUILD_ENFORCEMENT"
  ],
  "policyVersion": "1.1",
  "policySections": [
    {
      "policyGroups": [
        {
          "fieldName": "Fixed By",
          "values": [
            {
              "value": ".*"
            }
          ]
        },
        {
          "fieldName": "Severity",
          "values": [
            {
              "value": ">= IMPORTANT"
            }
          ]
        }
      ]
    }
  ]
}
`
