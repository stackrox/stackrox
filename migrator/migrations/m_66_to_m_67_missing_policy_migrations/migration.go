package m66tom67

import (
	"bytes"
	"embed"
	"fmt"
	"path/filepath"

	"github.com/golang/protobuf/jsonpb"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/migrations/policymigrationhelper"
	"github.com/stackrox/rox/migrator/types"
	bolt "go.etcd.io/bbolt"
)

var (
	migration = types.Migration{
		StartingSeqNum: 66,
		VersionAfter:   &storage.Version{SeqNum: 67},
		Run: func(databases *types.Databases) error {
			err := updatePolicies(databases.BoltDB)
			if err != nil {
				return errors.Wrap(err, "updating policies")
			}
			return nil
		},
	}

	// These are the policies as they were _before_ migration. If the policy in central doesn't match this, it won't get upgraded
	preMigrationPolicyFilesDir = "policies_to_compare"
	//go:embed policies_to_compare/*.json
	preMigrationPolicyFiles embed.FS

	policyBucketName = []byte("policies")

	istioExclusion       = &storage.Exclusion{Name: "Don't alert on istio-system namespace", Deployment: &storage.Exclusion_Deployment{Scope: &storage.Scope{Namespace: "istio-system"}}}
	kubeSystemExclusion  = &storage.Exclusion{Name: "Don't alert on kube-system namespace", Deployment: &storage.Exclusion_Deployment{Scope: &storage.Scope{Namespace: "kube-system"}}}
	scannerExclusion     = &storage.Exclusion{Name: "Don't alert on StackRox scanner", Deployment: &storage.Exclusion_Deployment{Name: "scanner", Scope: &storage.Scope{Namespace: "stackrox"}}}
	scannerV2Exclusion   = &storage.Exclusion{Name: "Don't alert on StackRox scanner-v2", Deployment: &storage.Exclusion_Deployment{Name: "scanner-v2", Scope: &storage.Scope{Namespace: "stackrox"}}}
	scannerV2DBExclusion = &storage.Exclusion{Name: "Don't alert on StackRox scanner-v2 database", Deployment: &storage.Exclusion_Deployment{Name: "scanner-v2-db", Scope: &storage.Scope{Namespace: "stackrox"}}}

	compareOnlySectionAndExclusions   = []policymigrationhelper.FieldComparator{policymigrationhelper.PolicySectionComparator, policymigrationhelper.ExclusionComparator}
	compareOnlySectionAndStringFields = []policymigrationhelper.FieldComparator{policymigrationhelper.PolicySectionComparator, policymigrationhelper.DescriptionComparator, policymigrationhelper.RationaleComparator, policymigrationhelper.RemediationComparator}
	compareAllFields                  = []policymigrationhelper.FieldComparator{policymigrationhelper.PolicySectionComparator, policymigrationhelper.ExclusionComparator, policymigrationhelper.DescriptionComparator, policymigrationhelper.RationaleComparator, policymigrationhelper.RemediationComparator}

	policiesToMigrate = map[string]policymigrationhelper.PolicyChanges{
		"2db9a279-2aec-4618-a85d-7f1bdf4911b1": {
			FieldsToCompare: compareOnlySectionAndExclusions,
			ToChange: policymigrationhelper.PolicyUpdates{
				ExclusionsToAdd: []*storage.Exclusion{istioExclusion},
			},
		},
		"2e90874a-3521-44de-85c6-5720f519a701": {
			FieldsToCompare: compareOnlySectionAndExclusions,
			ToChange: policymigrationhelper.PolicyUpdates{
				ExclusionsToAdd: []*storage.Exclusion{istioExclusion},
			},
		},
		"886c3c94-3a6a-4f2b-82fc-d6bf5a310840": {
			FieldsToCompare: compareOnlySectionAndExclusions,
			ToChange: policymigrationhelper.PolicyUpdates{
				ExclusionsToAdd: []*storage.Exclusion{istioExclusion},
			},
		},
		"fe9de18b-86db-44d5-a7c4-74173ccffe2e": {
			FieldsToCompare: compareOnlySectionAndExclusions,
			ToChange: policymigrationhelper.PolicyUpdates{
				ExclusionsToAdd: []*storage.Exclusion{istioExclusion},
			},
		},
		"014a03c6-9053-49b5-88ea-c1efcf19804f": {
			FieldsToCompare: compareOnlySectionAndExclusions,
			ToChange: policymigrationhelper.PolicyUpdates{
				ExclusionsToAdd: []*storage.Exclusion{istioExclusion},
			},
		},
		"880fd131-46f0-43d2-82c9-547f5aa7e043": {
			FieldsToCompare: compareOnlySectionAndExclusions,
			ToChange: policymigrationhelper.PolicyUpdates{
				ExclusionsToAdd: []*storage.Exclusion{istioExclusion},
			},
		},
		"550081a1-ad3a-4eab-a874-8eb68fab2bbd": {
			FieldsToCompare: compareOnlySectionAndExclusions,
			ToChange: policymigrationhelper.PolicyUpdates{
				ExclusionsToAdd: []*storage.Exclusion{istioExclusion},
			},
		},
		"8ac93556-4ad4-4220-a275-3f518db0ceb9": {
			FieldsToCompare: compareOnlySectionAndExclusions,
			ToChange: policymigrationhelper.PolicyUpdates{
				ExclusionsToAdd: []*storage.Exclusion{istioExclusion},
			},
		},
		"d3e480c1-c6de-4cd2-9006-9a3eb3ad36b6": {
			FieldsToCompare: compareOnlySectionAndExclusions,
			ToChange: policymigrationhelper.PolicyUpdates{
				ExclusionsToAdd: []*storage.Exclusion{istioExclusion},
			},
		},
		"1a498d97-0cc2-45f5-b32e-1f3cca6a3113": {
			FieldsToCompare: compareOnlySectionAndExclusions,
			ToChange: policymigrationhelper.PolicyUpdates{
				ExclusionsToAdd: []*storage.Exclusion{istioExclusion},
			},
		},
		"7760a5f3-bca4-4ca8-94a7-ad89edbc0e2c": {
			FieldsToCompare: compareOnlySectionAndExclusions,
			ToChange: policymigrationhelper.PolicyUpdates{
				ExclusionsToAdd: []*storage.Exclusion{
					kubeSystemExclusion,
					istioExclusion,
				},
				ExclusionsToRemove: []*storage.Exclusion{
					{Name: "Don't alert on Kube System Namespace", Deployment: &storage.Exclusion_Deployment{Scope: &storage.Scope{Namespace: "kube-system"}}},
				},
			},
		},
		"1913283f-ce3c-4134-84ef-195c4cd687ae": {
			FieldsToCompare: compareOnlySectionAndExclusions,
			ToChange: policymigrationhelper.PolicyUpdates{
				ExclusionsToRemove: []*storage.Exclusion{scannerV2Exclusion},
			},
		},
		"f4996314-c3d7-4553-803b-b24ce7febe48": {
			FieldsToCompare: compareOnlySectionAndExclusions,
			ToChange: policymigrationhelper.PolicyUpdates{
				ExclusionsToRemove: []*storage.Exclusion{scannerV2DBExclusion},
			},
		},
		"a788556c-9268-4f30-a114-d456f2380818": {
			FieldsToCompare: compareOnlySectionAndExclusions,
			ToChange: policymigrationhelper.PolicyUpdates{
				ExclusionsToRemove: []*storage.Exclusion{scannerV2DBExclusion},
			},
		},
		"f95ff08d-130a-465a-a27e-32ed1fb05555": {
			FieldsToCompare: compareOnlySectionAndExclusions,
			ToChange: policymigrationhelper.PolicyUpdates{
				ExclusionsToAdd:    []*storage.Exclusion{scannerExclusion},
				ExclusionsToRemove: []*storage.Exclusion{scannerV2Exclusion},
			},
		},
		"ddb7af9c-5ec1-45e1-a0cf-c36e3ef2b2ce": {
			FieldsToCompare: compareOnlySectionAndExclusions,
			ToChange: policymigrationhelper.PolicyUpdates{
				ExclusionsToAdd:    []*storage.Exclusion{scannerExclusion},
				ExclusionsToRemove: []*storage.Exclusion{scannerV2Exclusion},
			},
		},
		"74cfb824-2e65-46b7-b1b4-ba897e53af1f": {
			FieldsToCompare: compareAllFields,
			ToChange: policymigrationhelper.PolicyUpdates{
				ExclusionsToRemove: []*storage.Exclusion{scannerV2Exclusion, scannerV2DBExclusion},
				Remediation:        strPtr("Run `dpkg -r --force-all apt apt-get && dpkg -r --force-all debconf dpkg` in the image build for production containers."),
			},
		},
		"a9b9ecf7-9707-4e32-8b62-d03018ed454f": {
			FieldsToCompare: compareAllFields,
			ToChange: policymigrationhelper.PolicyUpdates{
				ExclusionsToAdd: []*storage.Exclusion{
					kubeSystemExclusion,
					istioExclusion,
				},
				ExclusionsToRemove: []*storage.Exclusion{
					{Name: "Don't alert on kube namespace", Deployment: &storage.Exclusion_Deployment{Scope: &storage.Scope{Namespace: "kube-system"}}},
				},
				Remediation: strPtr("Ensure that deployments do not mount sensitive host directories, or exclude this deployment if host mount is required."),
			},
		},
		"d7a275e1-1bba-47e7-92a1-42340c759883": {
			FieldsToCompare: compareOnlySectionAndStringFields,
			ToChange: policymigrationhelper.PolicyUpdates{
				Remediation: strPtr("Run `dpkg -r --force-all apt && dpkg -r --force-all debconf dpkg` in the image build for production containers. Change applications to no longer use package managers at runtime, if applicable."),
			},
		},
		"89cae2e6-0cb7-4329-8692-c2c3717c1237": {
			FieldsToCompare: compareOnlySectionAndStringFields,
			ToChange: policymigrationhelper.PolicyUpdates{
				Description: strPtr("This policy generates a violation for any process execution that is not explicitly allowed by a locked process baseline for a given container specification within a Kubernetes deployment."),
				Rationale:   strPtr("A locked process baseline communicates high confidence that execution of a process not included in the baseline positively indicates malicious activity."),
			},
		},
	}
)

func strPtr(s string) *string {
	return &s
}

func updatePolicies(db *bolt.DB) error {
	comparisonPolicies, err := getComparisonPoliciesFromFiles()
	if err != nil {
		return err
	}

	return policymigrationhelper.MigratePolicies(db, policiesToMigrate, comparisonPolicies)
}

func getComparisonPoliciesFromFiles() (map[string]*storage.Policy, error) {
	comparisonPolicies := make(map[string]*storage.Policy)
	for policyID := range policiesToMigrate {
		path := filepath.Join(preMigrationPolicyFilesDir, fmt.Sprintf("%s.json", policyID))
		contents, err := preMigrationPolicyFiles.ReadFile(path)
		if err != nil {
			return nil, errors.Wrapf(err, "unable to read file %s", path)
		}

		policy := new(storage.Policy)
		err = jsonpb.Unmarshal(bytes.NewReader(contents), policy)
		if err != nil {
			return nil, errors.Wrapf(err, "unable to unmarshal policy (%s) json", policyID)
		}
		comparisonPolicies[policyID] = policy
	}
	return comparisonPolicies, nil
}

func init() {
	migrations.MustRegisterMigration(migration)
}
