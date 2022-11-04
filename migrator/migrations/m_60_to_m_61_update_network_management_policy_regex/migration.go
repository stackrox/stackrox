package m60tom61

import (
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/bolthelpers"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/types"
	bolt "go.etcd.io/bbolt"
)

var (
	migration = types.Migration{
		StartingSeqNum: 60,
		VersionAfter:   &storage.Version{SeqNum: 61},
		Run: func(databases *types.Databases) error {
			err := updateNetworkManagementExecutionPolicy(databases.BoltDB)
			if err != nil {
				return errors.Wrap(err, "updating 'Network Management Execution' policy")
			}
			return nil
		},
	}

	policyBucketName  = []byte("policies")
	policyID          = "2361bb4c-4cf6-4997-bae6-825da6cf932e"
	oldPolicyCriteria = "([a-z])ip|ifrename|ethtool|ifconfig|([a-z])arp|ipmaddr|iptunnel|route|nameif|mii-tool"
	newPolicyCriteria = "ip|ifrename|ethtool|ifconfig|arp|ipmaddr|iptunnel|route|nameif|mii-tool"
	policyFieldName   = "Process Name"
	newExclusions     = []*storage.Exclusion{
		{
			Name: "Don't alert on kube-system namespace",
			Deployment: &storage.Exclusion_Deployment{
				Scope: &storage.Scope{
					Namespace: "kube-system",
				},
			},
		},
		{
			Name: "Don't alert on openshift namespaces",
			Deployment: &storage.Exclusion_Deployment{
				Scope: &storage.Scope{
					Namespace: "openshift-.*",
				},
			},
		},
	}
)

func updateNetworkManagementExecutionPolicy(db *bolt.DB) error {
	if exists, err := bolthelpers.BucketExists(db, policyBucketName); err != nil {
		return err
	} else if !exists {
		return nil
	}
	policyBucket := bolthelpers.TopLevelRef(db, policyBucketName)
	return policyBucket.Update(func(b *bolt.Bucket) error {
		v := b.Get([]byte(policyID))
		if v == nil {
			return nil
		}

		var policy storage.Policy
		if err := proto.Unmarshal(v, &policy); err != nil {
			return errors.Wrapf(err, "unmarshaling migrated policy with id %q", policyID)
		}

		// Update the policy only if it has not already been altered by customer.
		if len(policy.GetPolicySections()) != 1 {
			return nil
		}

		section := policy.GetPolicySections()[0]
		if len(section.GetPolicyGroups()) != 1 {
			return nil
		}

		group := section.GetPolicyGroups()[0]
		if group == nil {
			return nil
		}

		if group.GetFieldName() != policyFieldName {
			return nil
		}

		if len(group.GetValues()) != 1 {
			return nil
		}

		value := group.GetValues()[0]
		if value == nil {
			return nil
		}

		// Check that the value actually matches the old version.
		if value.GetValue() != oldPolicyCriteria {
			return nil
		}

		// Next check that the policy doesn't have exclusions and whitelists already
		if len(policy.GetExclusions()) > 0 {
			return nil
		}

		// Update to the newer policy criteria
		value.Value = newPolicyCriteria

		// Add new exclusion
		policy.Exclusions = append(policy.Exclusions, newExclusions...)

		policyBytes, err := proto.Marshal(&policy)
		if err != nil {
			return errors.Wrapf(err, "marshaling migrated policy %q with id %q", policy.GetName(), policy.GetId())
		}
		if err := b.Put([]byte(policyID), policyBytes); err != nil {
			return errors.Wrapf(err, "writing migrated policy with id %q to the store", policy.GetId())
		}
		return nil
	})
}

func init() {
	migrations.MustRegisterMigration(migration)
}
