package m63tom64

import (
	"reflect"

	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/bolthelpers"
	"github.com/stackrox/rox/migrator/log"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/types"
	bolt "go.etcd.io/bbolt"
)

type policyUpdate struct {
	existingNumExclusions int
	existingPolicyGroups  []*storage.PolicyGroup
	newExclusions         []*storage.Exclusion
}

var (
	migration = types.Migration{
		StartingSeqNum: 63,
		VersionAfter:   &storage.Version{SeqNum: 64},
		Run: func(databases *types.Databases) error {
			err := updatePoliciesWithOSExclusions(databases.BoltDB)
			if err != nil {
				return errors.Wrap(err, "updating policies to exclude OpenShift namespaces with cluster operators")
			}
			return nil
		},
	}

	policyBucketName = []byte("policies")

	osSdnExclusion = &storage.Exclusion{
		Name:       "Don't alert on openshift-sdn namespace",
		Deployment: &storage.Exclusion_Deployment{Scope: &storage.Scope{Namespace: "openshift-sdn"}},
	}
	osClusterCSIExclusion = &storage.Exclusion{
		Name:       "Don't alert on openshift-cluster-csi-drivers namespace",
		Deployment: &storage.Exclusion_Deployment{Scope: &storage.Scope{Namespace: "openshift-cluster-csi-drivers"}},
	}
	osKubeAPIServerExclusion = &storage.Exclusion{
		Name:       "Don't alert on openshift-kube-apiserver namespace",
		Deployment: &storage.Exclusion_Deployment{Scope: &storage.Scope{Namespace: "openshift-kube-apiserver"}},
	}
	osKubeSchedulerExclusion = &storage.Exclusion{
		Name:       "Don't alert on openshift-kube-scheduler namespace",
		Deployment: &storage.Exclusion_Deployment{Scope: &storage.Scope{Namespace: "openshift-kube-scheduler"}},
	}
	osOauthAPIServerExclusion = &storage.Exclusion{
		Name:       "Don't alert on openshift-oauth-apiserver namespace",
		Deployment: &storage.Exclusion_Deployment{Scope: &storage.Scope{Namespace: "openshift-oauth-apiserver"}},
	}
	osAPIServerExclusion = &storage.Exclusion{
		Name:       "Don't alert on openshift-apiserver namespace",
		Deployment: &storage.Exclusion_Deployment{Scope: &storage.Scope{Namespace: "openshift-apiserver"}},
	}
	osEtcdExclusion = &storage.Exclusion{
		Name:       "Don't alert on openshift-etcd namespace",
		Deployment: &storage.Exclusion_Deployment{Scope: &storage.Scope{Namespace: "openshift-etcd"}},
	}
	osKubeCtrlMgrExclusion = &storage.Exclusion{
		Name:       "Don't alert on openshift-kube-controller-manager namespace",
		Deployment: &storage.Exclusion_Deployment{Scope: &storage.Scope{Namespace: "openshift-kube-controller-manager"}},
	}
	osNetOperatorExclusion = &storage.Exclusion{
		Name:       "Don't alert on openshift-network-operator namespace",
		Deployment: &storage.Exclusion_Deployment{Scope: &storage.Scope{Namespace: "openshift-network-operator"}},
	}
	osDNSExclusion = &storage.Exclusion{
		Name:       "Don't alert on openshift-dns namespace",
		Deployment: &storage.Exclusion_Deployment{Scope: &storage.Scope{Namespace: "openshift-dns"}},
	}
	osClusterNodeTuningExclusion = &storage.Exclusion{
		Name:       "Don't alert on openshift-cluster-node-tuning-operator namespace",
		Deployment: &storage.Exclusion_Deployment{Scope: &storage.Scope{Namespace: "openshift-cluster-node-tuning-operator"}},
	}
	osMultusExclusion = &storage.Exclusion{
		Name:       "Don't alert on openshift-multus namespace",
		Deployment: &storage.Exclusion_Deployment{Scope: &storage.Scope{Namespace: "openshift-multus"}},
	}
	osMachineAPIExclusion = &storage.Exclusion{
		Name:       "Don't alert on openshift-machine-api namespace",
		Deployment: &storage.Exclusion_Deployment{Scope: &storage.Scope{Namespace: "openshift-machine-api"}},
	}
	osMachineConfigExclusion = &storage.Exclusion{
		Name:       "Don't alert on openshift-machine-config-operator namespace",
		Deployment: &storage.Exclusion_Deployment{Scope: &storage.Scope{Namespace: "openshift-machine-config-operator"}},
	}
	osClusterVerExclusion = &storage.Exclusion{
		Name:       "Don't alert on openshift-cluster-version namespace",
		Deployment: &storage.Exclusion_Deployment{Scope: &storage.Scope{Namespace: "openshift-cluster-version"}},
	}
	osImageRegistryExclusion = &storage.Exclusion{
		Name:       "Don't alert on node-ca dameonset in the openshift-image-registry namespace",
		Deployment: &storage.Exclusion_Deployment{Name: "node-ca", Scope: &storage.Scope{Namespace: "openshift-image-registry"}},
	}

	policiesToMigrate = map[string]policyUpdate{
		"880fd131-46f0-43d2-82c9-547f5aa7e043": { // iptables Execution
			existingNumExclusions: 2,
			existingPolicyGroups: []*storage.PolicyGroup{
				{FieldName: "Process Name", BooleanOperator: storage.BooleanOperator_OR, Negate: false, Values: []*storage.PolicyValue{{Value: "iptables"}}},
			},
			newExclusions: []*storage.Exclusion{
				osSdnExclusion,
			},
		},
		"ed8c7957-14de-40bc-aeab-d27ceeecfa7b": { // Iptables Executed in Privileged Container
			existingNumExclusions: 3,
			existingPolicyGroups: []*storage.PolicyGroup{
				{FieldName: "Privileged Container", BooleanOperator: storage.BooleanOperator_OR, Negate: false, Values: []*storage.PolicyValue{{Value: "true"}}},
				{FieldName: "Process Name", BooleanOperator: storage.BooleanOperator_OR, Negate: false, Values: []*storage.PolicyValue{{Value: "iptables"}}},
				{FieldName: "Process UID", BooleanOperator: storage.BooleanOperator_OR, Negate: false, Values: []*storage.PolicyValue{{Value: "0"}}},
			},
			newExclusions: []*storage.Exclusion{
				osSdnExclusion,
			},
		},
		"32d770b9-c6ba-4398-b48a-0c3e807644ed": { // Docker CIS 5.19: Ensure mount propagation mode is not enabled
			existingNumExclusions: 0,
			existingPolicyGroups: []*storage.PolicyGroup{
				{FieldName: "Mount Propagation", BooleanOperator: storage.BooleanOperator_OR, Negate: false, Values: []*storage.PolicyValue{{Value: "BIDIRECTIONAL"}}},
			},
			newExclusions: []*storage.Exclusion{
				osClusterCSIExclusion,
			},
		},
		"fe9de18b-86db-44d5-a7c4-74173ccffe2e": { // Privileged Container
			existingNumExclusions: 5,
			existingPolicyGroups: []*storage.PolicyGroup{
				{FieldName: "Privileged Container", BooleanOperator: storage.BooleanOperator_OR, Negate: false, Values: []*storage.PolicyValue{{Value: "true"}}},
			},
			newExclusions: []*storage.Exclusion{
				osKubeAPIServerExclusion,
				osEtcdExclusion,
				osAPIServerExclusion,
				osDNSExclusion,
				osClusterNodeTuningExclusion,
				osClusterCSIExclusion,
				osMachineConfigExclusion,
			},
		},
		"dce17697-1b72-49d2-b18a-05d893cd9368": { // Docker CIS 4.1: Ensure That a User for the Container Has Been Created
			existingNumExclusions: 2,
			existingPolicyGroups: []*storage.PolicyGroup{
				{FieldName: "Image User", BooleanOperator: storage.BooleanOperator_OR, Negate: false, Values: []*storage.PolicyValue{{Value: "0"}, {Value: "root"}}},
			},
			newExclusions: []*storage.Exclusion{
				osSdnExclusion,
				osKubeAPIServerExclusion,
				osEtcdExclusion,
				osAPIServerExclusion,
				osDNSExclusion,
				osClusterNodeTuningExclusion,
				osClusterCSIExclusion,
				osMachineConfigExclusion,
			},
		},
		"6226d4ad-7619-4a0b-a160-46373cfcee66": { // Docker CIS 5.9 and 5.20: Ensure that the host's network namespace is not shared
			existingNumExclusions: 1,
			existingPolicyGroups: []*storage.PolicyGroup{
				{FieldName: "Host Network", BooleanOperator: storage.BooleanOperator_OR, Negate: false, Values: []*storage.PolicyValue{{Value: "true"}}},
			},
			newExclusions: []*storage.Exclusion{
				osKubeAPIServerExclusion,
				osKubeSchedulerExclusion,
				osKubeCtrlMgrExclusion,
				osSdnExclusion,
				osNetOperatorExclusion,
				osMultusExclusion,
				osClusterVerExclusion,
				osImageRegistryExclusion,
			},
		},
		"a9b9ecf7-9707-4e32-8b62-d03018ed454f": { // Mounting Sensitive Host Directories
			existingNumExclusions: 4,
			existingPolicyGroups: []*storage.PolicyGroup{
				{FieldName: "Volume Source", BooleanOperator: storage.BooleanOperator_OR, Negate: false, Values: []*storage.PolicyValue{{Value: "(/etc/.*|/sys/.*|/dev/.*|/proc/.*|/var/.*)"}}},
			},
			newExclusions: []*storage.Exclusion{
				osKubeAPIServerExclusion,
				osKubeSchedulerExclusion,
				osEtcdExclusion,
				osKubeCtrlMgrExclusion,
				osOauthAPIServerExclusion,
				osAPIServerExclusion,
				osNetOperatorExclusion,
				osMachineAPIExclusion,
				osDNSExclusion,
				osClusterCSIExclusion,
				osClusterNodeTuningExclusion,
				osMultusExclusion,
				osImageRegistryExclusion,
				osSdnExclusion,
				osMachineConfigExclusion,
			},
		},
	}
)

func updatePoliciesWithOSExclusions(db *bolt.DB) error {
	if exists, err := bolthelpers.BucketExists(db, policyBucketName); err != nil {
		return err
	} else if !exists {
		return nil
	}

	return db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(policyBucketName)

		// Migrate and update policies one by one. Abort the transaction, and hence
		// the migration, in case of any error.
		for policyID, updateDetails := range policiesToMigrate {
			v := bucket.Get([]byte(policyID))
			if v == nil {
				log.WriteToStderrf("no policy exists for ID %s in policy category migration. Continuing", policyID)
				continue
			}

			var policy storage.Policy
			if err := proto.Unmarshal(v, &policy); err != nil {
				// Unable to recover, so abort transaction
				return errors.Wrapf(err, "unmarshaling migrated policy with id %q", policyID)
			}

			// Update the policy only if it has not already been altered by customer.
			if len(policy.PolicySections) != 1 {
				log.WriteToStderrf("policy ID %s has already been altered. Will not update.", policyID)
				continue
			}

			section := policy.PolicySections[0]
			if !reflect.DeepEqual(section.PolicyGroups, updateDetails.existingPolicyGroups) {
				log.WriteToStderrf("policy ID %s has already been altered. Will not update.", policyID)
				continue
			}

			// Next check that the policy doesn't have any extra exclusions already
			if len(policy.Exclusions) > updateDetails.existingNumExclusions {
				log.WriteToStderrf("policy ID %s has already been altered. Will not update.", policyID)
				continue
			}

			// Add new exclusion
			policy.Exclusions = append(policy.Exclusions, updateDetails.newExclusions...)

			policyBytes, err := proto.Marshal(&policy)
			if err != nil {
				return errors.Wrapf(err, "marshaling migrated policy %q with id %q", policy.GetName(), policy.GetId())
			}
			if err := bucket.Put([]byte(policyID), policyBytes); err != nil {
				return errors.Wrapf(err, "writing migrated policy with id %q to the store", policy.GetId())
			}
		}

		return nil
	})
}

func init() {
	migrations.MustRegisterMigration(migration)
}
