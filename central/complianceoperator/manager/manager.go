package manager

import (
	"context"

	"github.com/ComplianceAsCode/compliance-operator/pkg/apis/compliance/v1alpha1"
	checkResultsDatastore "github.com/stackrox/rox/central/complianceoperator/checkresults/datastore"
	profileDatastore "github.com/stackrox/rox/central/complianceoperator/profiles/datastore"
	rulesDatastore "github.com/stackrox/rox/central/complianceoperator/rules/datastore"
	scansDatastore "github.com/stackrox/rox/central/complianceoperator/scans/datastore"
	scanSettingBindingDatastore "github.com/stackrox/rox/central/complianceoperator/scansettingbinding/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
)

var (
	allAccessCtx = sac.WithAllAccess(context.Background())
)

// Manager helps manage the dynamic profiles from the compliance operator
//
//go:generate mockgen-wrapper
type Manager interface {
	AddProfile(profile *storage.ComplianceOperatorProfile) error
	DeleteProfile(profile *storage.ComplianceOperatorProfile) error

	AddRule(rule *storage.ComplianceOperatorRule) error
	DeleteRule(rule *storage.ComplianceOperatorRule) error

	AddScan(scan *storage.ComplianceOperatorScan) error
	DeleteScan(scan *storage.ComplianceOperatorScan) error

	GetMachineConfigs(clusterID string) (map[string][]string, error)
}

type managerImpl struct {
	profiles            profileDatastore.DataStore
	scanSettingBindings scanSettingBindingDatastore.DataStore
	scans               scansDatastore.DataStore
	rules               rulesDatastore.DataStore
	results             checkResultsDatastore.DataStore
}

// NewManager returns a new manager of compliance operator resources
func NewManager(profiles profileDatastore.DataStore, scans scansDatastore.DataStore, scanSettingBindings scanSettingBindingDatastore.DataStore, rules rulesDatastore.DataStore, results checkResultsDatastore.DataStore) (Manager, error) {
	mgr := &managerImpl{
		profiles:            profiles,
		scans:               scans,
		scanSettingBindings: scanSettingBindings,
		rules:               rules,
		results:             results,
	}
	return mgr, nil
}

func (m *managerImpl) AddProfile(profile *storage.ComplianceOperatorProfile) error {
	return m.profiles.Upsert(allAccessCtx, profile)
}

func (m *managerImpl) DeleteProfile(deletedProfile *storage.ComplianceOperatorProfile) error {
	return m.profiles.Delete(allAccessCtx, deletedProfile.GetId())
}

func (m *managerImpl) AddScan(scan *storage.ComplianceOperatorScan) error {
	return m.scans.Upsert(allAccessCtx, scan)
}

func (m *managerImpl) DeleteScan(scan *storage.ComplianceOperatorScan) error {
	return m.scans.Delete(allAccessCtx, scan.GetId())
}

func (m *managerImpl) GetMachineConfigs(clusterID string) (map[string][]string, error) {
	profileIDsToNames := make(map[string]string)
	walkFn := func() error {
		profileIDsToNames = make(map[string]string)
		return m.profiles.Walk(allAccessCtx, func(profile *storage.ComplianceOperatorProfile) error {
			if profile.GetClusterId() == clusterID && profile.GetAnnotations()[v1alpha1.ProductTypeAnnotation] == string(v1alpha1.ScanTypeNode) {
				profileIDsToNames[profile.GetProfileId()] = profile.GetName()
			}
			return nil
		})
	}
	ctx := context.Background()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := walkFn(); err != nil {
		return nil, err
	}

	profilesToScan := make(map[string][]string)
	walkFn = func() error {
		profilesToScan = make(map[string][]string)
		return m.scans.Walk(allAccessCtx, func(scan *storage.ComplianceOperatorScan) error {
			if scan.GetClusterId() != clusterID {
				return nil
			}
			if profileName, ok := profileIDsToNames[scan.GetProfileId()]; ok {
				profilesToScan[profileName] = append(profilesToScan[profileName], scan.GetName())
			}
			return nil
		})
	}
	if err := walkFn(); err != nil {
		return nil, err
	}
	return profilesToScan, nil
}

func (m *managerImpl) AddRule(rule *storage.ComplianceOperatorRule) error {
	return m.rules.Upsert(allAccessCtx, rule)
}

func (m *managerImpl) DeleteRule(rule *storage.ComplianceOperatorRule) error {
	return m.rules.Delete(allAccessCtx, rule.GetId())
}
