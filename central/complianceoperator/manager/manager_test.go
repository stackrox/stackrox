//go:build sql_integration

package manager

import (
	"testing"

	"github.com/ComplianceAsCode/compliance-operator/pkg/apis/compliance/v1alpha1"
	"github.com/stackrox/rox/central/compliance/datastore/mocks"
	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/central/compliance/standards"
	"github.com/stackrox/rox/central/compliance/standards/metadata"
	checkResultsDatastore "github.com/stackrox/rox/central/complianceoperator/checkresults/datastore"
	checkResultsStore "github.com/stackrox/rox/central/complianceoperator/checkresults/store/postgres"
	profileDatastore "github.com/stackrox/rox/central/complianceoperator/profiles/datastore"
	profileStore "github.com/stackrox/rox/central/complianceoperator/profiles/store/postgres"
	rulesDatastore "github.com/stackrox/rox/central/complianceoperator/rules/datastore"
	rulesStore "github.com/stackrox/rox/central/complianceoperator/rules/store/postgres"
	scansDatastore "github.com/stackrox/rox/central/complianceoperator/scans/datastore"
	scansStore "github.com/stackrox/rox/central/complianceoperator/scans/store/postgres"
	scanSettingBindingDatastore "github.com/stackrox/rox/central/complianceoperator/scansettingbinding/datastore"
	scanSettingBindingStore "github.com/stackrox/rox/central/complianceoperator/scansettingbinding/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func newManager(t *testing.T) *managerImpl {
	registry, err := standards.NewRegistry(framework.RegistrySingleton(), metadata.AllStandards...)
	require.NoError(t, err)

	db := pgtest.ForT(t)
	prof := profileStore.New(db)
	ssb := scanSettingBindingStore.New(db)
	rules := rulesStore.New(db)
	rulesDS, err := rulesDatastore.NewDatastore(rules)
	require.NoError(t, err)
	scans := scansStore.New(db)
	scansDS := scansDatastore.NewDatastore(scans)

	checks := checkResultsStore.New(db)

	ctrl := gomock.NewController(t)
	compliance := mocks.NewMockDataStore(ctrl)

	mgr, err := NewManager(registry, profileDatastore.NewDatastore(prof), scansDS, scanSettingBindingDatastore.NewDatastore(ssb), rulesDS, checkResultsDatastore.NewDatastore(checks), compliance)
	require.NoError(t, err)

	return mgr.(*managerImpl)
}

func TestAddProfile(t *testing.T) {
	mgr := newManager(t)

	rule1Name := "rule1"
	rule1 := &storage.ComplianceOperatorRule{
		Id:   uuid.NewV4().String(),
		Name: "rule1-ext",
		Annotations: map[string]string{
			v1alpha1.RuleIDAnnotationKey: rule1Name,
		},
		ClusterId: "cluster1",
		Title:     "title1",
	}
	rule2Name := "rule2"
	rule2 := &storage.ComplianceOperatorRule{
		Id:   uuid.NewV4().String(),
		Name: "rule2-ext",
		Annotations: map[string]string{
			v1alpha1.RuleIDAnnotationKey: rule2Name,
		},
		ClusterId: "cluster1",
		Title:     "title2",
	}

	assert.NoError(t, mgr.AddRule(rule1))
	assert.NoError(t, mgr.AddRule(rule2))

	initialProfile := &storage.ComplianceOperatorProfile{
		Id:        uuid.NewV4().String(),
		Name:      "profile1",
		ClusterId: "cluster1",
		Rules: []*storage.ComplianceOperatorProfile_Rule{
			{
				Name: rule1.GetName(),
			},
			{
				Name: rule2.GetName(),
			},
		},
	}

	// Base case, no existing profiles
	assert.NoError(t, mgr.AddProfile(initialProfile))
	control1 := mgr.registry.Control(standards.BuildQualifiedID("profile1", rule1Name))
	assert.Equal(t, rule1Name, control1.GetName())
	assert.Equal(t, rule1.GetTitle(), control1.GetDescription())

	control2 := mgr.registry.Control(standards.BuildQualifiedID("profile1", rule2Name))
	assert.Equal(t, rule2Name, control2.GetName())
	assert.Equal(t, rule2.GetTitle(), control2.GetDescription())

	// Update same profile and verify existing controls
	rule2.Title = "new rule2 title"
	assert.NoError(t, mgr.AddRule(rule2))
	assert.NoError(t, mgr.AddProfile(initialProfile))

	control2 = mgr.registry.Control(standards.BuildQualifiedID("profile1", rule2Name))
	assert.Equal(t, rule2Name, control2.GetName())
	assert.Equal(t, rule2.GetTitle(), control2.GetDescription())

	// Remove rule two from profile 1 and verify that the control is also removed
	initialProfile.Rules = []*storage.ComplianceOperatorProfile_Rule{
		{
			Name: rule1.GetName(),
		},
	}
	assert.NoError(t, mgr.AddProfile(initialProfile))
	assert.Nil(t, mgr.registry.Control(standards.BuildQualifiedID("profile1", rule2Name)))

	rule3Name := "rule3"
	rule3 := &storage.ComplianceOperatorRule{
		Id:   uuid.NewV4().String(),
		Name: "rule3-ext",
		Annotations: map[string]string{
			v1alpha1.RuleIDAnnotationKey: rule3Name,
		},
		ClusterId: "cluster2",
		Title:     "title3",
	}
	duplicateNamedProfile := &storage.ComplianceOperatorProfile{
		Id:        uuid.NewV4().String(),
		Name:      initialProfile.GetName(),
		ClusterId: "cluster2",
		Rules: []*storage.ComplianceOperatorProfile_Rule{
			{
				Name: rule1.GetName(),
			},
			{
				Name: rule2.GetName(),
			},
			{
				Name: rule3.GetName(),
			},
		},
	}
	assert.NoError(t, mgr.AddRule(rule3))
	assert.NoError(t, mgr.AddProfile(duplicateNamedProfile))

	control3 := mgr.registry.Control(standards.BuildQualifiedID("profile1", rule3Name))
	assert.Equal(t, rule3Name, control3.GetName())
	assert.Equal(t, rule3.GetTitle(), control3.GetDescription())
}

func TestDeleteProfile(t *testing.T) {
	mgr := newManager(t)

	mgr.compliance.(*mocks.MockDataStore).EXPECT().ClearAggregationResults(allAccessCtx).AnyTimes()

	rule1Name := "rule1"
	rule1 := &storage.ComplianceOperatorRule{
		Id:   uuid.NewV4().String(),
		Name: "rule1-ext",
		Annotations: map[string]string{
			v1alpha1.RuleIDAnnotationKey: rule1Name,
		},
		ClusterId: "cluster1",
		Title:     "title1",
	}
	rule2Name := "rule2"
	rule2 := &storage.ComplianceOperatorRule{
		Id:   uuid.NewV4().String(),
		Name: "rule2-ext",
		Annotations: map[string]string{
			v1alpha1.RuleIDAnnotationKey: rule2Name,
		},
		ClusterId: "cluster1",
		Title:     "title2",
	}

	assert.NoError(t, mgr.AddRule(rule1))
	assert.NoError(t, mgr.AddRule(rule2))

	initialProfile := &storage.ComplianceOperatorProfile{
		Id:        uuid.NewV4().String(),
		Name:      "profile1",
		ClusterId: "cluster1",
		Rules: []*storage.ComplianceOperatorProfile_Rule{
			{
				Name: rule1.GetName(),
			},
			{
				Name: rule2.GetName(),
			},
		},
	}

	// Base case, add and delete without any other profiles
	assert.NoError(t, mgr.AddProfile(initialProfile))
	control1 := mgr.registry.Control(standards.BuildQualifiedID("profile1", rule1Name))
	assert.NotNil(t, control1)

	control2 := mgr.registry.Control(standards.BuildQualifiedID("profile1", rule2Name))
	assert.NotNil(t, control2)

	// Delete profile and verify controls are removed
	assert.NoError(t, mgr.DeleteProfile(initialProfile))
	control1 = mgr.registry.Control(standards.BuildQualifiedID("profile1", rule1Name))
	assert.Nil(t, control1)

	control2 = mgr.registry.Control(standards.BuildQualifiedID("profile1", rule2Name))
	assert.Nil(t, control2)

	// Add profile back and then add profile with same name
	assert.NoError(t, mgr.AddProfile(initialProfile))
	control1 = mgr.registry.Control(standards.BuildQualifiedID("profile1", rule1Name))
	assert.NotNil(t, control1)

	control2 = mgr.registry.Control(standards.BuildQualifiedID("profile1", rule2Name))
	assert.NotNil(t, control2)

	updatedProfile := initialProfile.Clone()
	updatedProfile.Id = uuid.NewV4().String()
	// Add the updated profile and delete the original profile. The controls should still exist
	assert.NoError(t, mgr.AddProfile(updatedProfile))

	assert.NoError(t, mgr.DeleteProfile(updatedProfile))
	control1 = mgr.registry.Control(standards.BuildQualifiedID("profile1", rule1Name))
	assert.NotNil(t, control1)

	control2 = mgr.registry.Control(standards.BuildQualifiedID("profile1", rule2Name))
	assert.NotNil(t, control2)

	// Add rule3 and check its existence, then delete the updated profile and ensure rule3 is removed
	rule3Name := "rule3"
	rule3 := &storage.ComplianceOperatorRule{
		Id:   uuid.NewV4().String(),
		Name: "rule3-ext",
		Annotations: map[string]string{
			v1alpha1.RuleIDAnnotationKey: rule3Name,
		},
		ClusterId: "cluster2",
		Title:     "title3",
	}
	assert.NoError(t, mgr.AddRule(rule3))
	updatedProfile.Rules = append(updatedProfile.Rules, &storage.ComplianceOperatorProfile_Rule{Name: rule3.GetName()})
	assert.NoError(t, mgr.AddProfile(updatedProfile))

	control3 := mgr.registry.Control(standards.BuildQualifiedID("profile1", rule3Name))
	assert.NotNil(t, control3)
	assert.NoError(t, mgr.DeleteProfile(updatedProfile))

	// Control 1 and 2 should still exist, but control 3 should not after the updated profile is removed as it is the only one referencing it
	control1 = mgr.registry.Control(standards.BuildQualifiedID("profile1", rule1Name))
	assert.NotNil(t, control1)

	control2 = mgr.registry.Control(standards.BuildQualifiedID("profile1", rule2Name))
	assert.NotNil(t, control2)

	control3 = mgr.registry.Control(standards.BuildQualifiedID("profile1", rule3Name))
	assert.Nil(t, control3)
}

func TestIsStandardActiveFunctions(t *testing.T) {
	mgr := newManager(t)

	assert.False(t, mgr.IsStandardActive("random"))
	assert.False(t, mgr.IsStandardActiveForCluster("random", "thisdoesntmatter"))

	dockerID, err := mgr.registry.GetCISKubernetesStandardID()
	assert.NoError(t, err)
	assert.True(t, mgr.IsStandardActive(dockerID))
	assert.True(t, mgr.IsStandardActiveForCluster(dockerID, "thisdoesntmatter"))

	profile := &storage.ComplianceOperatorProfile{
		Id:        uuid.NewV4().String(),
		Name:      "dynamicprofile",
		ClusterId: "clusterid",
	}
	err = mgr.AddProfile(profile)
	assert.NoError(t, err)

	// Name is the standard ID
	assert.False(t, mgr.IsStandardActive(profile.GetName()))
	assert.False(t, mgr.IsStandardActiveForCluster(profile.GetName(), "clusterid"))

	scanSettingBinding := &storage.ComplianceOperatorScanSettingBinding{
		Id:        uuid.NewV4().String(),
		Name:      "dynamicprofile-binding",
		ClusterId: "clusterid",
		Profiles: []*storage.ComplianceOperatorScanSettingBinding_Profile{
			{
				Name: "dynamicprofile",
			},
		},
	}
	assert.NoError(t, mgr.scanSettingBindings.Upsert(allAccessCtx, scanSettingBinding))
	assert.True(t, mgr.IsStandardActive(profile.GetName()))

	// Check wrong is standard active for cluster
	assert.False(t, mgr.IsStandardActiveForCluster(profile.GetName(), "notacluster"))
	assert.True(t, mgr.IsStandardActiveForCluster(profile.GetName(), "clusterid"))
}

func TestAddRule(t *testing.T) {
	mgr := newManager(t)

	rule1Name := "rule1"
	rule1 := &storage.ComplianceOperatorRule{
		Id:   uuid.NewV4().String(),
		Name: "rule1-ext",
		Annotations: map[string]string{
			v1alpha1.RuleIDAnnotationKey: rule1Name,
		},
		ClusterId: "cluster1",
		Title:     "title1",
	}

	rule2Name := "rule2"
	rule2 := &storage.ComplianceOperatorRule{
		Id:   uuid.NewV4().String(),
		Name: "rule2-ext",
		Annotations: map[string]string{
			v1alpha1.RuleIDAnnotationKey: rule2Name,
		},
		ClusterId: "cluster1",
		Title:     "title2",
	}
	// Add a rule when there are no profiles, shouldn't do anything
	assert.NoError(t, mgr.AddRule(rule1))
	assert.NoError(t, mgr.AddRule(rule1))
	assert.Nil(t, mgr.registry.Control(standards.BuildQualifiedID("profile1", rule1Name)))

	initialProfile := &storage.ComplianceOperatorProfile{
		Id:        uuid.NewV4().String(),
		Name:      "profile1",
		ClusterId: "cluster1",
		Rules: []*storage.ComplianceOperatorProfile_Rule{
			{
				Name: rule1.GetName(),
			},
			{
				Name: rule2.GetName(),
			},
		},
	}
	// Insert profile where rule1 exists and rule2 does not
	assert.NoError(t, mgr.AddProfile(initialProfile))
	assert.NotNil(t, mgr.registry.Control(standards.BuildQualifiedID("profile1", rule1Name)))
	assert.Nil(t, mgr.registry.Control(standards.BuildQualifiedID("profile1", rule2Name)))

	assert.NoError(t, mgr.AddRule(rule2))
	assert.NoError(t, mgr.AddRule(rule2))

	assert.NotNil(t, mgr.registry.Control(standards.BuildQualifiedID("profile1", rule1Name)))
	assert.NotNil(t, mgr.registry.Control(standards.BuildQualifiedID("profile1", rule2Name)))
}

func TestDeleteRule(t *testing.T) {
	mgr := newManager(t)

	rule1Name := "rule1"
	rule1 := &storage.ComplianceOperatorRule{
		Id:   uuid.NewV4().String(),
		Name: "rule1-ext",
		Annotations: map[string]string{
			v1alpha1.RuleIDAnnotationKey: rule1Name,
		},
		ClusterId: "cluster1",
		Title:     "title1",
	}

	rule2Name := "rule2"
	rule2 := &storage.ComplianceOperatorRule{
		Id:   uuid.NewV4().String(),
		Name: "rule2-ext",
		Annotations: map[string]string{
			v1alpha1.RuleIDAnnotationKey: rule2Name,
		},
		ClusterId: "cluster1",
		Title:     "title2",
	}
	assert.NoError(t, mgr.AddRule(rule1))
	assert.NoError(t, mgr.AddRule(rule2))

	initialProfile := &storage.ComplianceOperatorProfile{
		Id:        uuid.NewV4().String(),
		Name:      "profile1",
		ClusterId: "cluster1",
		Rules: []*storage.ComplianceOperatorProfile_Rule{
			{
				Name: rule1.GetName(),
			},
			{
				Name: rule2.GetName(),
			},
		},
	}
	// Insert profile where rule1 exists and rule2 does not
	assert.NoError(t, mgr.AddProfile(initialProfile))
	assert.NotNil(t, mgr.registry.Control(standards.BuildQualifiedID("profile1", rule1Name)))
	assert.NotNil(t, mgr.registry.Control(standards.BuildQualifiedID("profile1", rule2Name)))

	assert.NoError(t, mgr.DeleteRule(rule1))

	assert.Nil(t, mgr.registry.Control(standards.BuildQualifiedID("profile1", rule1Name)))
	assert.NotNil(t, mgr.registry.Control(standards.BuildQualifiedID("profile1", rule2Name)))

	assert.NoError(t, mgr.DeleteRule(rule2))
	assert.Nil(t, mgr.registry.Control(standards.BuildQualifiedID("profile1", rule2Name)))
}

func TestGetMachineConfigs(t *testing.T) {
	mgr := newManager(t)

	result, err := mgr.GetMachineConfigs("")
	assert.NoError(t, err)
	assert.Len(t, result, 0)

	initialProfile := &storage.ComplianceOperatorProfile{
		Id:        uuid.NewV4().String(),
		Name:      "profile1",
		ProfileId: "profile-id",
		ClusterId: "cluster1",
		Annotations: map[string]string{
			v1alpha1.ProductTypeAnnotation: string(v1alpha1.ScanTypeNode),
		},
		Rules: []*storage.ComplianceOperatorProfile_Rule{},
	}
	assert.NoError(t, mgr.AddProfile(initialProfile))
	result, err = mgr.GetMachineConfigs("cluster1")
	assert.NoError(t, err)
	assert.Len(t, result, 0)

	scan := &storage.ComplianceOperatorScan{
		Id:        uuid.NewV4().String(),
		Name:      "scan",
		ClusterId: "cluster1",
		ProfileId: "profile-id",
	}
	assert.NoError(t, mgr.AddScan(scan))
	result, err = mgr.GetMachineConfigs("cluster1")
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, result["profile1"], []string{"scan"})

	scan2 := &storage.ComplianceOperatorScan{
		Id:        uuid.NewV4().String(),
		Name:      "scan2",
		ClusterId: "cluster1",
		ProfileId: "profile-id",
	}
	assert.NoError(t, mgr.AddScan(scan2))
	result, err = mgr.GetMachineConfigs("cluster1")
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.ElementsMatch(t, result["profile1"], []string{"scan", "scan2"})
}
