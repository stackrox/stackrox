package m65tom66

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

var (
	migration = types.Migration{
		StartingSeqNum: 65,
		VersionAfter:   &storage.Version{SeqNum: 66},
		Run: func(databases *types.Databases) error {
			err := updatePolicies(databases.BoltDB)
			if err != nil {
				return errors.Wrap(err, "updating policies")
			}
			return nil
		},
	}

	policyBucketName = []byte("policies")

	// Kubernetes Dashboard Deployed
	k8sDashPolicyID             = "0ac267ae-9128-42c7-b15e-0e926844aa2f"
	k8sDashNewCriteria          = "r/.*kubernetesui/dashboard.*"
	k8sDashExistingPolicyGroups = []*storage.PolicyGroup{
		{FieldName: "Image Remote", BooleanOperator: storage.BooleanOperator_OR, Negate: false, Values: []*storage.PolicyValue{{Value: "r/.*kubernetes-dashboard-amd64.*"}}},
	}

	// Curl in Image
	curlPolicyID             = "1913283f-ce3c-4134-84ef-195c4cd687ae"
	curlExistingRemediation  = "Use your package manager's \"remove\" command to remove curl from the image build for production containers."
	curlNewRemediation       = "Use your package manager's \"remove\", \"purge\" or \"erase\" command to remove curl from the image build for production containers. Ensure that any configuration files are also removed."
	curlExistingPolicyGroups = []*storage.PolicyGroup{
		{FieldName: "Image Component", BooleanOperator: storage.BooleanOperator_OR, Negate: false, Values: []*storage.PolicyValue{{Value: "curl="}}},
	}

	// Iptables Executed in Privileged Container
	iptablesPolicyID          = "ed8c7957-14de-40bc-aeab-d27ceeecfa7b"
	iptablesExclusionToRemove = &storage.Exclusion{
		Name:       "Don't alert on stackrox namespace",
		Deployment: &storage.Exclusion_Deployment{Scope: &storage.Scope{Namespace: "stackrox"}},
	}
	iptablesExistingPolicyGroups = []*storage.PolicyGroup{
		{FieldName: "Privileged Container", BooleanOperator: storage.BooleanOperator_OR, Negate: false, Values: []*storage.PolicyValue{{Value: "true"}}},
		{FieldName: "Process Name", BooleanOperator: storage.BooleanOperator_OR, Negate: false, Values: []*storage.PolicyValue{{Value: "iptables"}}},
		{FieldName: "Process UID", BooleanOperator: storage.BooleanOperator_OR, Negate: false, Values: []*storage.PolicyValue{{Value: "0"}}},
	}
)

func updatePolicies(db *bolt.DB) error {
	if exists, err := bolthelpers.BucketExists(db, policyBucketName); err != nil {
		return err
	} else if !exists {
		return nil
	}

	return db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(policyBucketName)

		// 1st policy: Kubernetes Dashboard Deployed
		if err := updateK8sDashPolicy(bucket); err != nil {
			return err
		}

		// 2nd policy: Curl in Image
		if err := updateCurlPolicy(bucket); err != nil {
			return err
		}

		// 3rd policy: Iptables Executed in Privileged Container
		if err := updateIptablesPolicy(bucket); err != nil {
			return err
		}

		return nil
	})
}

func updateK8sDashPolicy(bucket *bolt.Bucket) error {
	v := bucket.Get([]byte(k8sDashPolicyID))
	if v == nil {
		log.WriteToStderrf("no policy exists for ID %s in policy migration. Continuing", k8sDashPolicyID)
		return nil
	}

	var policy storage.Policy
	if err := proto.Unmarshal(v, &policy); err != nil {
		// Unable to recover, so abort transaction
		return errors.Wrapf(err, "unmarshaling migrated policy with id %q", k8sDashPolicyID)
	}

	// Update the policy only if it has not already been altered by customer.
	section := getPolicySectionIfUnmodified(&policy, k8sDashExistingPolicyGroups)
	if section == nil {
		log.WriteToStderrf("policy ID %s has already been altered. Will not update.", k8sDashPolicyID)
		return nil
	}

	// Next check that the policy doesn't have any extra exclusions
	if len(policy.Exclusions) > 0 {
		return nil
	}

	section.PolicyGroups[0].Values[0].Value = k8sDashNewCriteria
	if err := putPolicy(&policy, bucket); err != nil {
		return err
	}

	return nil
}

func updateCurlPolicy(bucket *bolt.Bucket) error {
	v := bucket.Get([]byte(curlPolicyID))
	if v == nil {
		log.WriteToStderrf("no policy exists for ID %s in policy migration. Continuing", curlPolicyID)
		return nil
	}

	var policy storage.Policy
	if err := proto.Unmarshal(v, &policy); err != nil {
		// Unable to recover, so abort transaction
		return errors.Wrapf(err, "unmarshaling migrated policy with id %q", curlPolicyID)
	}

	// Update the policy only if it has not already been altered by customer.
	if policy.Remediation != curlExistingRemediation {
		log.WriteToStderrf("policy ID %s has already been altered. Will not update.", curlPolicyID)
		return nil
	}

	section := getPolicySectionIfUnmodified(&policy, curlExistingPolicyGroups)
	if section == nil {
		log.WriteToStderrf("policy ID %s has already been altered. Will not update.", curlPolicyID)
		return nil
	}

	policy.Remediation = curlNewRemediation
	if err := putPolicy(&policy, bucket); err != nil {
		return err
	}

	return nil
}

func updateIptablesPolicy(bucket *bolt.Bucket) error {
	v := bucket.Get([]byte(iptablesPolicyID))
	if v == nil {
		log.WriteToStderrf("no policy exists for ID %s in policy migration. Continuing", iptablesPolicyID)
		return nil
	}

	var policy storage.Policy
	if err := proto.Unmarshal(v, &policy); err != nil {
		// Unable to recover, so abort transaction
		return errors.Wrapf(err, "unmarshaling migrated policy with id %q", iptablesPolicyID)
	}

	// Update the policy only if it has not already been altered by customer.
	section := getPolicySectionIfUnmodified(&policy, iptablesExistingPolicyGroups)
	if section == nil {
		log.WriteToStderrf("policy ID %s has already been altered. Will not update.", iptablesPolicyID)
		return nil
	}

	if !removeExclusion(&policy, iptablesExclusionToRemove) {
		log.WriteToStderrf("policy ID %s has already been altered. Will not update.", iptablesPolicyID)
		return nil
	}

	if err := putPolicy(&policy, bucket); err != nil {
		return err
	}

	return nil
}

func removeExclusion(policy *storage.Policy, exclusionToRemove *storage.Exclusion) bool {
	exclusions := policy.Exclusions
	for i, exclusion := range exclusions {
		if reflect.DeepEqual(exclusion, exclusionToRemove) {
			policy.Exclusions = append(exclusions[:i], exclusions[i+1:]...)
			return true
		}
	}
	return false
}

func getPolicySectionIfUnmodified(policy *storage.Policy, existingPolicyGroups []*storage.PolicyGroup) *storage.PolicySection {
	if len(policy.PolicySections) != 1 {
		return nil
	}

	section := policy.PolicySections[0]
	if !reflect.DeepEqual(section.PolicyGroups, existingPolicyGroups) {
		return nil
	}

	return section
}

func putPolicy(policy *storage.Policy, bucket *bolt.Bucket) error {
	policyBytes, err := proto.Marshal(policy)
	if err != nil {
		return errors.Wrapf(err, "marshaling migrated policy %q with id %q", policy.GetName(), policy.GetId())
	}
	if err := bucket.Put([]byte(policy.GetId()), policyBytes); err != nil {
		return errors.Wrapf(err, "writing migrated policy with id %q to the store", policy.GetId())
	}
	return nil
}

func init() {
	migrations.MustRegisterMigration(migration)
}
