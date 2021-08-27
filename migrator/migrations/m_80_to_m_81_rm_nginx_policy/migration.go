package m80tom81

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

const (
	nginxPolicyID = "5a90a571-58e7-4ed5-a2fa-2dbe83e649ba"

	nginxPolicyJSON = `{
  "id": "5a90a571-58e7-4ed5-a2fa-2dbe83e649ba",
  "name": "DockerHub NGINX 1.10",
  "description": "Alert on deployments with nginx:1.10 image from 'docker.io'",
  "rationale": "This is an example of policy that you could create. nginx:1.10 has many vulnerabilities.",
  "remediation": "Migrate to the latest stable release of NGINX.",
  "categories": [
    "DevOps Best Practices",
    "Security Best Practices"
  ],
  "lifecycleStages": [
    "DEPLOY"
  ],
  "severity": "MEDIUM_SEVERITY",
  "policyVersion": "1.1",
  "policySections": [
    {
      "policyGroups": [
        {
          "fieldName": "Image Registry",
          "values": [
            {
              "value": "docker.io"
            }
          ]
        },
        {
          "fieldName": "Image Remote",
          "values": [
            {
              "value": "r/.*nginx.*"
            }
          ]
        },
        {
          "fieldName": "Image Tag",
          "values": [
            {
              "value": "1.10"
            }
          ]
        }
      ]
    }
  ],
  "criteriaLocked": true,
  "mitreVectorsLocked": true
}`
)

var (
	migration = types.Migration{
		StartingSeqNum: 80,
		VersionAfter:   storage.Version{SeqNum: 81},
		Run: func(databases *types.Databases) error {
			err := rmNginxPolicy(databases.BoltDB)
			if err != nil {
				return errors.Wrap(err, "removing default system policies 'DockerHub NGINX 1.10'")
			}
			return nil
		},
	}

	policyBucket = []byte("policies")
)

func init() {
	migrations.MustRegisterMigration(migration)
}

func rmNginxPolicy(db *bolt.DB) error {
	nginxPolicy := &storage.Policy{}
	if err := jsonpb.Unmarshal(strings.NewReader(nginxPolicyJSON), nginxPolicy); err != nil {
		return errors.Wrap(err, "unmarshalling 'DockerHub NGINX 1.10' policy JSON")
	}

	err := db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(policyBucket)
		if bucket == nil {
			return errors.Errorf("bucket %q not found", policyBucket)
		}

		val := bucket.Get([]byte(nginxPolicy.GetId()))
		if val == nil {
			return nil
		}

		storedPolicy := &storage.Policy{}
		if err := proto.Unmarshal(val, storedPolicy); err != nil {
			return errors.Wrapf(err, "unmarshaling policy with ID %q", nginxPolicy.GetId())
		}

		if !proto.Equal(storedPolicy, nginxPolicy) {
			return nil
		}

		if err := bucket.Delete([]byte(nginxPolicy.GetId())); err != nil {
			return errors.Wrapf(err, "removing policy %s", nginxPolicy.GetId())
		}
		return nil
	})
	return err
}
