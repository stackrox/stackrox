package manager

import (
	"testing"

	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/central/compliance/standards"
	"github.com/stackrox/rox/central/compliance/standards/metadata"
	profileDatastore "github.com/stackrox/rox/central/complianceoperator/profiles/datastore"
	profileStore "github.com/stackrox/rox/central/complianceoperator/profiles/store"
	rulesDatastore "github.com/stackrox/rox/central/complianceoperator/rules/datastore"
	rulesStore "github.com/stackrox/rox/central/complianceoperator/rules/store"
	scanSettingBindingDatastore "github.com/stackrox/rox/central/complianceoperator/scansettingbinding/datastore"
	scanSettingBindingStore "github.com/stackrox/rox/central/complianceoperator/scansettingbinding/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/testutils/rocksdbtest"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newManager(t *testing.T) *managerImpl {
	registry, err := standards.NewRegistry(nil, framework.RegistrySingleton(), metadata.AllStandards...)
	require.NoError(t, err)

	db := rocksdbtest.RocksDBForT(t)
	prof, err := profileStore.New(db)
	require.NoError(t, err)

	ssb, err := scanSettingBindingStore.New(db)
	require.NoError(t, err)

	rules, err := rulesStore.New(db)
	require.NoError(t, err)

	rulesDS, err := rulesDatastore.NewDatastore(rules)
	require.NoError(t, err)

	mgr, err := NewManager(registry, profileDatastore.NewDatastore(prof), scanSettingBindingDatastore.NewDatastore(ssb), rulesDS)
	require.NoError(t, err)

	return mgr.(*managerImpl)
}

func TestAddProfile(t *testing.T) {
	mgr := newManager(t)

	rule1 := &storage.ComplianceOperatorRule{
		Id:        uuid.NewV4().String(),
		Name:      "rule1",
		ClusterId: "cluster1",
		Title:     "title1",
	}
	rule2 := &storage.ComplianceOperatorRule{
		Id:        uuid.NewV4().String(),
		Name:      "rule2",
		ClusterId: "cluster1",
		Title:     "title2",
	}

	assert.NoError(t, mgr.rules.Upsert(allAccessCtx, rule1))
	assert.NoError(t, mgr.rules.Upsert(allAccessCtx, rule2))

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
	assert.NoError(t, mgr.profiles.Upsert(allAccessCtx, initialProfile))
	assert.NoError(t, mgr.AddProfile(initialProfile))
	control1 := mgr.registry.Control(standards.BuildQualifiedID("profile1", "rule1"))
	assert.Equal(t, rule1.GetName(), control1.GetName())
	assert.Equal(t, rule1.GetTitle(), control1.GetDescription())

	control2 := mgr.registry.Control(standards.BuildQualifiedID("profile1", "rule2"))
	assert.Equal(t, rule2.GetName(), control2.GetName())
	assert.Equal(t, rule2.GetTitle(), control2.GetDescription())

	// Update same profile and verify existing controls
	rule2.Title = "new rule2 title"
	assert.NoError(t, mgr.rules.Upsert(allAccessCtx, rule2))
	assert.NoError(t, mgr.AddProfile(initialProfile))

	control2 = mgr.registry.Control(standards.BuildQualifiedID("profile1", "rule2"))
	assert.Equal(t, rule2.GetName(), control2.GetName())
	assert.Equal(t, rule2.GetTitle(), control2.GetDescription())

	// Remove rule two from profile 1 and verify that the control is also removed
	initialProfile.Rules = []*storage.ComplianceOperatorProfile_Rule{
		{
			Name: rule1.GetName(),
		},
	}
	assert.NoError(t, mgr.profiles.Upsert(allAccessCtx, initialProfile))
	assert.NoError(t, mgr.AddProfile(initialProfile))
	assert.Nil(t, mgr.registry.Control(standards.BuildQualifiedID("profile1", "rule2")))

	rule3 := &storage.ComplianceOperatorRule{
		Id:        uuid.NewV4().String(),
		Name:      "rule3",
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
	assert.NoError(t, mgr.rules.Upsert(allAccessCtx, rule3))
	assert.NoError(t, mgr.profiles.Upsert(allAccessCtx, duplicateNamedProfile))
	assert.NoError(t, mgr.AddProfile(duplicateNamedProfile))

	control3 := mgr.registry.Control(standards.BuildQualifiedID("profile1", "rule3"))
	assert.Equal(t, rule3.GetName(), control3.GetName())
	assert.Equal(t, rule3.GetTitle(), control3.GetDescription())
}

func TestDeleteProfile(t *testing.T) {
	mgr := newManager(t)

	rule1 := &storage.ComplianceOperatorRule{
		Id:        uuid.NewV4().String(),
		Name:      "rule1",
		ClusterId: "cluster1",
		Title:     "title1",
	}
	rule2 := &storage.ComplianceOperatorRule{
		Id:        uuid.NewV4().String(),
		Name:      "rule2",
		ClusterId: "cluster1",
		Title:     "title2",
	}

	assert.NoError(t, mgr.rules.Upsert(allAccessCtx, rule1))
	assert.NoError(t, mgr.rules.Upsert(allAccessCtx, rule2))

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
	assert.NoError(t, mgr.profiles.Upsert(allAccessCtx, initialProfile))
	assert.NoError(t, mgr.AddProfile(initialProfile))
	control1 := mgr.registry.Control(standards.BuildQualifiedID("profile1", "rule1"))
	assert.NotNil(t, control1)

	control2 := mgr.registry.Control(standards.BuildQualifiedID("profile1", "rule2"))
	assert.NotNil(t, control2)

	// Delete profile and verify controls are removed
	assert.NoError(t, mgr.profiles.Delete(allAccessCtx, initialProfile.GetId()))
	assert.NoError(t, mgr.DeleteProfile(initialProfile))
	control1 = mgr.registry.Control(standards.BuildQualifiedID("profile1", "rule1"))
	assert.Nil(t, control1)

	control2 = mgr.registry.Control(standards.BuildQualifiedID("profile1", "rule2"))
	assert.Nil(t, control2)

	// Add profile back and then add profile with same name
	assert.NoError(t, mgr.profiles.Upsert(allAccessCtx, initialProfile))
	assert.NoError(t, mgr.AddProfile(initialProfile))
	control1 = mgr.registry.Control(standards.BuildQualifiedID("profile1", "rule1"))
	assert.NotNil(t, control1)

	control2 = mgr.registry.Control(standards.BuildQualifiedID("profile1", "rule2"))
	assert.NotNil(t, control2)

	updatedProfile := initialProfile.Clone()
	updatedProfile.Id = uuid.NewV4().String()
	// Add the updated profile and delete the original profile. The controls should still exist
	assert.NoError(t, mgr.profiles.Upsert(allAccessCtx, updatedProfile))
	assert.NoError(t, mgr.AddProfile(updatedProfile))

	assert.NoError(t, mgr.profiles.Delete(allAccessCtx, updatedProfile.GetId()))
	assert.NoError(t, mgr.DeleteProfile(updatedProfile))
	control1 = mgr.registry.Control(standards.BuildQualifiedID("profile1", "rule1"))
	assert.NotNil(t, control1)

	control2 = mgr.registry.Control(standards.BuildQualifiedID("profile1", "rule2"))
	assert.NotNil(t, control2)

	// Add rule3 and check its existence, then delete the updated profile and ensure rule3 is removed
	rule3 := &storage.ComplianceOperatorRule{
		Id:        uuid.NewV4().String(),
		Name:      "rule3",
		ClusterId: "cluster2",
		Title:     "title3",
	}
	assert.NoError(t, mgr.rules.Upsert(allAccessCtx, rule3))
	updatedProfile.Rules = append(updatedProfile.Rules, &storage.ComplianceOperatorProfile_Rule{Name: rule3.GetName()})
	assert.NoError(t, mgr.profiles.Upsert(allAccessCtx, updatedProfile))
	assert.NoError(t, mgr.AddProfile(updatedProfile))

	control3 := mgr.registry.Control(standards.BuildQualifiedID("profile1", "rule3"))
	assert.NotNil(t, control3)
	assert.NoError(t, mgr.profiles.Delete(allAccessCtx, updatedProfile.GetId()))
	assert.NoError(t, mgr.DeleteProfile(updatedProfile))

	// Control 1 and 2 should still exist, but control 3 should not after the updated profile is removed as it is the only one referencing it
	control1 = mgr.registry.Control(standards.BuildQualifiedID("profile1", "rule1"))
	assert.NotNil(t, control1)

	control2 = mgr.registry.Control(standards.BuildQualifiedID("profile1", "rule2"))
	assert.NotNil(t, control2)

	control3 = mgr.registry.Control(standards.BuildQualifiedID("profile1", "rule3"))
	assert.Nil(t, control3)
}

func TestIsStandardActiveFunctions(t *testing.T) {
	mgr := newManager(t)

	assert.False(t, mgr.IsStandardActive("random"))
	assert.False(t, mgr.IsStandardActiveForCluster("random", "thisdoesntmatter"))

	dockerID, err := mgr.registry.GetCISDockerStandardID()
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
