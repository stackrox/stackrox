package tests

import (
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	deploymentEnvsMocks "github.com/stackrox/rox/central/deploymentenvs/mocks"
	"github.com/stackrox/rox/central/license/manager"
	licenseMgrMocks "github.com/stackrox/rox/central/license/manager/mocks"
	licenseStoreMocks "github.com/stackrox/rox/central/license/store/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	licenseproto "github.com/stackrox/rox/generated/shared/license"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	validatorMocks "github.com/stackrox/rox/pkg/license/validator/mocks"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type managerTestSuite struct {
	suite.Suite

	mockCtrl              *gomock.Controller
	mockStore             *licenseStoreMocks.MockStore
	mockValidator         *validatorMocks.MockValidator
	mockListener          *licenseMgrMocks.MockLicenseEventListener
	mockDeploymentEnvsMgr *deploymentEnvsMocks.MockManager

	mgr manager.LicenseManager
}

func TestManager(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(managerTestSuite))
}

func (s *managerTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockStore = licenseStoreMocks.NewMockStore(s.mockCtrl)
	s.mockValidator = validatorMocks.NewMockValidator(s.mockCtrl)
	s.mockListener = licenseMgrMocks.NewMockLicenseEventListener(s.mockCtrl)
	s.mockDeploymentEnvsMgr = deploymentEnvsMocks.NewMockManager(s.mockCtrl)

	s.mgr = manager.New(s.mockStore, s.mockValidator, s.mockDeploymentEnvsMgr)

	s.mockDeploymentEnvsMgr.EXPECT().RegisterListener(gomock.Any()).Times(1)
	s.mockDeploymentEnvsMgr.EXPECT().GetDeploymentEnvironmentsByClusterID(gomock.Any()).AnyTimes().Return(map[string][]string{})
}

func (s *managerTestSuite) TearDownTest() {
	time.Sleep(100 * time.Millisecond)
	s.True(concurrency.WaitWithTimeout(s.mgr.Stop(), time.Second))
	s.mockCtrl.Finish()
}

func (s *managerTestSuite) TestInitializeEmpty() {
	s.mockStore.EXPECT().ListLicenseKeys().Return(nil, nil)
	s.mockListener.EXPECT().OnInitialize(s.mgr, gomock.Nil())
	activeLicense, err := s.mgr.Initialize(s.mockListener)
	s.Nil(activeLicense)
	s.Equal(v1.Metadata_NONE, s.mgr.GetLicenseStatus())
	s.NoError(err)
}

func (s *managerTestSuite) TestInitializeWithValidAndSelected() {
	s.mockStore.EXPECT().ListLicenseKeys().Return([]*storage.StoredLicenseKey{
		{
			LicenseKey: "KEY1",
			LicenseId:  "license1",
			Selected:   false,
		},
		{
			LicenseKey: "KEY2",
			LicenseId:  "license2",
			Selected:   true,
		},
	}, nil)

	s.mockValidator.EXPECT().ValidateLicenseKey("KEY1").Return(&licenseproto.License{
		Metadata: &licenseproto.License_Metadata{
			Id: "license1",
		},
		Restrictions: &licenseproto.License_Restrictions{
			NotValidBefore:                     protoconv.ConvertTimeToTimestamp(time.Now()),
			NotValidAfter:                      protoconv.ConvertTimeToTimestamp(time.Now().Add(10 * time.Second)),
			AllowOffline:                       true,
			NoNodeRestriction:                  true,
			NoBuildFlavorRestriction:           true,
			NoDeploymentEnvironmentRestriction: true,
		},
	}, nil)

	license2 := &licenseproto.License{
		Metadata: &licenseproto.License_Metadata{
			Id: "license2",
		},
		Restrictions: &licenseproto.License_Restrictions{
			NotValidBefore:                     protoconv.ConvertTimeToTimestamp(time.Now()),
			NotValidAfter:                      protoconv.ConvertTimeToTimestamp(time.Now().Add(10 * time.Second)),
			AllowOffline:                       true,
			NoNodeRestriction:                  true,
			NoBuildFlavorRestriction:           true,
			NoDeploymentEnvironmentRestriction: true,
		},
	}

	s.mockValidator.EXPECT().ValidateLicenseKey("KEY2").Return(license2, nil)

	s.mockListener.EXPECT().OnInitialize(s.mgr, license2)

	activeLicense, err := s.mgr.Initialize(s.mockListener)
	s.NoError(err)

	s.Equal("license2", activeLicense.GetMetadata().GetId())
	s.Equal(v1.Metadata_VALID, s.mgr.GetLicenseStatus())
}

func (s *managerTestSuite) TestInitializeWithInvalidSelected() {
	s.mockStore.EXPECT().ListLicenseKeys().Return([]*storage.StoredLicenseKey{
		{
			LicenseKey: "KEY1",
			LicenseId:  "license1",
			Selected:   true,
		},
		{
			LicenseKey: "KEY2",
			LicenseId:  "license2",
			Selected:   false,
		},
	}, nil)

	s.mockValidator.EXPECT().ValidateLicenseKey("KEY1").Return(&licenseproto.License{
		Metadata: &licenseproto.License_Metadata{
			Id: "license1",
		},
		Restrictions: &licenseproto.License_Restrictions{
			NotValidBefore:                     protoconv.ConvertTimeToTimestamp(time.Now().Add(-20 * time.Second)),
			NotValidAfter:                      protoconv.ConvertTimeToTimestamp(time.Now().Add(-10 * time.Second)),
			AllowOffline:                       true,
			NoNodeRestriction:                  true,
			NoBuildFlavorRestriction:           true,
			NoDeploymentEnvironmentRestriction: true,
		},
	}, nil)

	license2 := &licenseproto.License{
		Metadata: &licenseproto.License_Metadata{
			Id: "license2",
		},
		Restrictions: &licenseproto.License_Restrictions{
			NotValidBefore:                     protoconv.ConvertTimeToTimestamp(time.Now()),
			NotValidAfter:                      protoconv.ConvertTimeToTimestamp(time.Now().Add(10 * time.Second)),
			AllowOffline:                       true,
			NoNodeRestriction:                  true,
			NoBuildFlavorRestriction:           true,
			NoDeploymentEnvironmentRestriction: true,
		},
	}

	s.mockValidator.EXPECT().ValidateLicenseKey("KEY2").Return(license2, nil)

	s.mockListener.EXPECT().OnInitialize(s.mgr, license2)

	s.mockStore.EXPECT().UpsertLicenseKeys(
		testutils.AssertionMatcher(
			assert.ElementsMatch,
			[]*storage.StoredLicenseKey{
				{
					LicenseId:  "license1",
					LicenseKey: "KEY1",
					Selected:   false,
				},
				{
					LicenseId:  "license2",
					LicenseKey: "KEY2",
					Selected:   true,
				},
			}))

	activeLicense, err := s.mgr.Initialize(s.mockListener)
	s.NoError(err)

	s.Equal("license2", activeLicense.GetMetadata().GetId())
	s.Equal(v1.Metadata_VALID, s.mgr.GetLicenseStatus())
}

func (s *managerTestSuite) TestInitializeWithNoneSelected() {
	s.mockStore.EXPECT().ListLicenseKeys().Return([]*storage.StoredLicenseKey{
		{
			LicenseKey: "KEY1",
			LicenseId:  "license1",
			Selected:   false,
		},
		{
			LicenseKey: "KEY2",
			LicenseId:  "license2",
			Selected:   false,
		},
	}, nil)

	s.mockValidator.EXPECT().ValidateLicenseKey("KEY1").Return(&licenseproto.License{
		Metadata: &licenseproto.License_Metadata{
			Id: "license1",
		},
		Restrictions: &licenseproto.License_Restrictions{
			NotValidBefore:                     protoconv.ConvertTimeToTimestamp(time.Now()),
			NotValidAfter:                      protoconv.ConvertTimeToTimestamp(time.Now().Add(10 * time.Second)),
			AllowOffline:                       true,
			NoNodeRestriction:                  true,
			NoBuildFlavorRestriction:           true,
			NoDeploymentEnvironmentRestriction: true,
		},
	}, nil)

	license2 := &licenseproto.License{
		Metadata: &licenseproto.License_Metadata{
			Id: "license2",
		},
		Restrictions: &licenseproto.License_Restrictions{
			NotValidBefore:                     protoconv.ConvertTimeToTimestamp(time.Now()),
			NotValidAfter:                      protoconv.ConvertTimeToTimestamp(time.Now().Add(20 * time.Second)),
			AllowOffline:                       true,
			NoNodeRestriction:                  true,
			NoBuildFlavorRestriction:           true,
			NoDeploymentEnvironmentRestriction: true,
		},
	}

	s.mockValidator.EXPECT().ValidateLicenseKey("KEY2").Return(license2, nil)

	s.mockListener.EXPECT().OnInitialize(s.mgr, license2)

	s.mockStore.EXPECT().UpsertLicenseKeys(
		[]*storage.StoredLicenseKey{
			{
				LicenseKey: "KEY2",
				LicenseId:  "license2",
				Selected:   true,
			},
		})

	activeLicense, err := s.mgr.Initialize(s.mockListener)
	s.NoError(err)

	s.Equal("license2", activeLicense.GetMetadata().GetId())
	s.Equal(v1.Metadata_VALID, s.mgr.GetLicenseStatus())
}

func (s *managerTestSuite) TestLicenseSwitchOnExpiration() {
	s.mockStore.EXPECT().ListLicenseKeys().Return([]*storage.StoredLicenseKey{
		{
			LicenseKey: "KEY1",
			LicenseId:  "license1",
			Selected:   true,
		},
		{
			LicenseKey: "KEY2",
			LicenseId:  "license2",
			Selected:   false,
		},
	}, nil)

	license1 := &licenseproto.License{
		Metadata: &licenseproto.License_Metadata{
			Id: "license1",
		},
		Restrictions: &licenseproto.License_Restrictions{
			NotValidBefore:                     protoconv.ConvertTimeToTimestamp(time.Now()),
			NotValidAfter:                      protoconv.ConvertTimeToTimestamp(time.Now().Add(1 * time.Second)),
			AllowOffline:                       true,
			NoNodeRestriction:                  true,
			NoBuildFlavorRestriction:           true,
			NoDeploymentEnvironmentRestriction: true,
		},
	}

	s.mockValidator.EXPECT().ValidateLicenseKey("KEY1").Return(license1, nil)

	s.mockValidator.EXPECT().ValidateLicenseKey("KEY2").Return(&licenseproto.License{
		Metadata: &licenseproto.License_Metadata{
			Id: "license2",
		},
		Restrictions: &licenseproto.License_Restrictions{
			NotValidBefore:                     protoconv.ConvertTimeToTimestamp(time.Now()),
			NotValidAfter:                      protoconv.ConvertTimeToTimestamp(time.Now().Add(20 * time.Second)),
			AllowOffline:                       true,
			NoNodeRestriction:                  true,
			NoBuildFlavorRestriction:           true,
			NoDeploymentEnvironmentRestriction: true,
		},
	}, nil)

	s.mockListener.EXPECT().OnInitialize(s.mgr, license1)

	activeLicense, err := s.mgr.Initialize(s.mockListener)
	s.NoError(err)

	s.Equal("license1", activeLicense.GetMetadata().GetId())
	s.Equal(v1.Metadata_VALID, s.mgr.GetLicenseStatus())

	s.mockListener.EXPECT().OnActiveLicenseChanged(
		testutils.PredMatcher("new license is license 2 and valid", func(l *v1.LicenseInfo) bool {
			if l.GetLicense().GetMetadata().GetId() != "license2" {
				return false
			}
			return l.GetStatus() == v1.LicenseInfo_VALID
		}),
		testutils.PredMatcher("old license is license 1 and expired", func(l *v1.LicenseInfo) bool {
			if l.GetLicense().GetMetadata().GetId() != "license1" {
				return false
			}
			return l.GetStatus() == v1.LicenseInfo_EXPIRED
		}))

	s.mockStore.EXPECT().UpsertLicenseKeys(testutils.AssertionMatcher(assert.ElementsMatch, []*storage.StoredLicenseKey{
		{
			LicenseKey: "KEY1",
			LicenseId:  "license1",
			Selected:   false,
		},
		{
			LicenseKey: "KEY2",
			LicenseId:  "license2",
			Selected:   true,
		},
	}))
	time.Sleep(2 * time.Second)

	newActiveLicense := s.mgr.GetActiveLicense()
	s.Equal("license2", newActiveLicense.GetMetadata().GetId())

	s.Equal(v1.Metadata_VALID, s.mgr.GetLicenseStatus())
}

func (s *managerTestSuite) TestLicenseSwitchOffOnExpiration() {
	s.mockStore.EXPECT().ListLicenseKeys().Return([]*storage.StoredLicenseKey{
		{
			LicenseKey: "KEY1",
			LicenseId:  "license1",
			Selected:   true,
		},
		{
			LicenseKey: "KEY2",
			LicenseId:  "license2",
			Selected:   false,
		},
	}, nil)

	license1 := &licenseproto.License{
		Metadata: &licenseproto.License_Metadata{
			Id: "license1",
		},
		Restrictions: &licenseproto.License_Restrictions{
			NotValidBefore:                     protoconv.ConvertTimeToTimestamp(time.Now()),
			NotValidAfter:                      protoconv.ConvertTimeToTimestamp(time.Now().Add(1 * time.Second)),
			AllowOffline:                       true,
			NoNodeRestriction:                  true,
			NoBuildFlavorRestriction:           true,
			NoDeploymentEnvironmentRestriction: true,
		},
	}

	s.mockValidator.EXPECT().ValidateLicenseKey("KEY1").Return(license1, nil)

	s.mockValidator.EXPECT().ValidateLicenseKey("KEY2").Return(&licenseproto.License{
		Metadata: &licenseproto.License_Metadata{
			Id: "license2",
		},
		Restrictions: &licenseproto.License_Restrictions{
			NotValidBefore:                     protoconv.ConvertTimeToTimestamp(time.Now()),
			NotValidAfter:                      protoconv.ConvertTimeToTimestamp(time.Now().Add(20 * time.Second)),
			AllowOffline:                       true,
			NoNodeRestriction:                  true,
			NoBuildFlavorRestriction:           false,
			BuildFlavors:                       []string{"some-really-weird-build-flavor"},
			NoDeploymentEnvironmentRestriction: true,
		},
	}, nil)

	s.mockListener.EXPECT().OnInitialize(s.mgr, license1)

	activeLicense, err := s.mgr.Initialize(s.mockListener)
	s.NoError(err)

	s.Equal("license1", activeLicense.GetMetadata().GetId())
	s.Equal(v1.Metadata_VALID, s.mgr.GetLicenseStatus())

	s.mockListener.EXPECT().OnActiveLicenseChanged(
		nil,
		testutils.PredMatcher("old license is license 1 and expired", func(l *v1.LicenseInfo) bool {
			if l.GetLicense().GetMetadata().GetId() != "license1" {
				return false
			}
			return l.GetStatus() == v1.LicenseInfo_EXPIRED
		}))

	s.mockStore.EXPECT().UpsertLicenseKeys([]*storage.StoredLicenseKey{
		{
			LicenseKey: "KEY1",
			LicenseId:  "license1",
			Selected:   false,
		},
	})
	time.Sleep(2 * time.Second)

	newActiveLicense := s.mgr.GetActiveLicense()
	s.Nil(newActiveLicense)
	s.Equal(v1.Metadata_EXPIRED, s.mgr.GetLicenseStatus())
}

func (s *managerTestSuite) TestLicenseActivatedWhenValid() {
	s.mockStore.EXPECT().ListLicenseKeys().Return([]*storage.StoredLicenseKey{
		{
			LicenseKey: "KEY1",
			LicenseId:  "license1",
			Selected:   false,
		},
	}, nil)

	s.mockValidator.EXPECT().ValidateLicenseKey("KEY1").Return(&licenseproto.License{
		Metadata: &licenseproto.License_Metadata{
			Id: "license1",
		},
		Restrictions: &licenseproto.License_Restrictions{
			NotValidBefore:                     protoconv.ConvertTimeToTimestamp(time.Now().Add(1 * time.Second)),
			NotValidAfter:                      protoconv.ConvertTimeToTimestamp(time.Now().Add(10 * time.Second)),
			AllowOffline:                       true,
			NoNodeRestriction:                  true,
			NoBuildFlavorRestriction:           true,
			NoDeploymentEnvironmentRestriction: true,
		},
	}, nil)

	s.mockListener.EXPECT().OnInitialize(s.mgr, gomock.Nil())

	activeLicense, err := s.mgr.Initialize(s.mockListener)
	s.NoError(err)
	s.Nil(activeLicense)
	s.Equal(v1.Metadata_INVALID, s.mgr.GetLicenseStatus())

	s.mockListener.EXPECT().OnActiveLicenseChanged(
		testutils.PredMatcher("new license is license 1 and is valid", func(l *v1.LicenseInfo) bool {
			if l.GetLicense().GetMetadata().GetId() != "license1" {
				return false
			}
			return l.GetStatus() == v1.LicenseInfo_VALID
		}),
		nil)

	s.mockStore.EXPECT().UpsertLicenseKeys([]*storage.StoredLicenseKey{
		{
			LicenseKey: "KEY1",
			LicenseId:  "license1",
			Selected:   true,
		},
	})
	time.Sleep(2 * time.Second)

	newActiveLicense := s.mgr.GetActiveLicense()
	s.Equal("license1", newActiveLicense.GetMetadata().GetId())
	s.Equal(v1.Metadata_VALID, s.mgr.GetLicenseStatus())
}

func (s *managerTestSuite) TestLicenseActivatedWhenValidAdded() {
	s.mockStore.EXPECT().ListLicenseKeys().Return(nil, nil)

	s.mockValidator.EXPECT().ValidateLicenseKey("KEY1").Return(&licenseproto.License{
		Metadata: &licenseproto.License_Metadata{
			Id: "license1",
		},
		Restrictions: &licenseproto.License_Restrictions{
			NotValidBefore:                     protoconv.ConvertTimeToTimestamp(time.Now().Add(-1 * time.Second)),
			NotValidAfter:                      protoconv.ConvertTimeToTimestamp(time.Now().Add(10 * time.Second)),
			AllowOffline:                       true,
			NoNodeRestriction:                  true,
			NoBuildFlavorRestriction:           true,
			NoDeploymentEnvironmentRestriction: true,
		},
	}, nil)

	s.mockListener.EXPECT().OnInitialize(s.mgr, gomock.Nil())

	activeLicense, err := s.mgr.Initialize(s.mockListener)
	s.NoError(err)
	s.Nil(activeLicense)
	s.Equal(v1.Metadata_NONE, s.mgr.GetLicenseStatus())

	s.mockListener.EXPECT().OnActiveLicenseChanged(
		testutils.PredMatcher("new license is license 1 and is valid", func(l *v1.LicenseInfo) bool {
			if l.GetLicense().GetMetadata().GetId() != "license1" {
				return false
			}
			return l.GetStatus() == v1.LicenseInfo_VALID
		}),
		nil)

	s.mockStore.EXPECT().UpsertLicenseKeys([]*storage.StoredLicenseKey{
		{
			LicenseKey: "KEY1",
			LicenseId:  "license1",
			Selected:   true,
		},
	})
	addedLicense, err := s.mgr.AddLicenseKey("KEY1", false)
	s.Require().NoError(err)

	s.Equal("license1", addedLicense.GetLicense().GetMetadata().GetId())
	s.Equal(v1.LicenseInfo_VALID, addedLicense.GetStatus())

	newActiveLicense := s.mgr.GetActiveLicense()
	s.Equal(addedLicense.GetLicense(), newActiveLicense)
	s.Equal(v1.Metadata_VALID, s.mgr.GetLicenseStatus())
}

func (s *managerTestSuite) TestLicenseNotReplacedWithActivateFalse() {
	s.mockStore.EXPECT().ListLicenseKeys().Return([]*storage.StoredLicenseKey{
		{
			LicenseKey: "KEY1",
			LicenseId:  "license1",
			Selected:   true,
		},
	}, nil)

	license1 := &licenseproto.License{
		Metadata: &licenseproto.License_Metadata{
			Id: "license1",
		},
		Restrictions: &licenseproto.License_Restrictions{
			NotValidBefore:                     protoconv.ConvertTimeToTimestamp(time.Now()),
			NotValidAfter:                      protoconv.ConvertTimeToTimestamp(time.Now().Add(1 * time.Hour)),
			AllowOffline:                       true,
			NoNodeRestriction:                  true,
			NoBuildFlavorRestriction:           true,
			NoDeploymentEnvironmentRestriction: true,
		},
	}

	s.mockValidator.EXPECT().ValidateLicenseKey("KEY1").Return(license1, nil)

	s.mockListener.EXPECT().OnInitialize(s.mgr, license1)

	activeLicense, err := s.mgr.Initialize(s.mockListener)
	s.NoError(err)

	s.Equal("license1", activeLicense.GetMetadata().GetId())
	s.Equal(v1.Metadata_VALID, s.mgr.GetLicenseStatus())

	license2 := &licenseproto.License{
		Metadata: &licenseproto.License_Metadata{
			Id: "license2",
		},
		Restrictions: &licenseproto.License_Restrictions{
			NotValidBefore:                     protoconv.ConvertTimeToTimestamp(time.Now()),
			NotValidAfter:                      protoconv.ConvertTimeToTimestamp(time.Now().Add(1 * time.Hour)),
			AllowOffline:                       true,
			NoNodeRestriction:                  true,
			NoBuildFlavorRestriction:           true,
			NoDeploymentEnvironmentRestriction: true,
		},
	}

	s.mockValidator.EXPECT().ValidateLicenseKey("KEY2").Return(license2, nil)
	s.mockStore.EXPECT().UpsertLicenseKeys([]*storage.StoredLicenseKey{
		{
			LicenseKey: "KEY2",
			LicenseId:  "license2",
		},
	})

	addedLicense, err := s.mgr.AddLicenseKey("KEY2", false)
	s.NoError(err)
	s.Equal("license2", addedLicense.GetLicense().GetMetadata().GetId())
	s.Equal(v1.LicenseInfo_VALID, addedLicense.GetStatus())
	s.False(addedLicense.GetActive())

	activeLicense = s.mgr.GetActiveLicense()
	s.Equal("license1", activeLicense.GetMetadata().GetId())
	s.Equal(v1.Metadata_VALID, s.mgr.GetLicenseStatus())
}

func (s *managerTestSuite) TestLicenseIsReplacedWithActivateTrue() {
	s.mockStore.EXPECT().ListLicenseKeys().Return([]*storage.StoredLicenseKey{
		{
			LicenseKey: "KEY1",
			LicenseId:  "license1",
			Selected:   true,
		},
	}, nil)

	license1 := &licenseproto.License{
		Metadata: &licenseproto.License_Metadata{
			Id: "license1",
		},
		Restrictions: &licenseproto.License_Restrictions{
			NotValidBefore:                     protoconv.ConvertTimeToTimestamp(time.Now()),
			NotValidAfter:                      protoconv.ConvertTimeToTimestamp(time.Now().Add(1 * time.Hour)),
			AllowOffline:                       true,
			NoNodeRestriction:                  true,
			NoBuildFlavorRestriction:           true,
			NoDeploymentEnvironmentRestriction: true,
		},
	}

	s.mockValidator.EXPECT().ValidateLicenseKey("KEY1").Return(license1, nil)

	s.mockListener.EXPECT().OnInitialize(s.mgr, license1)

	activeLicense, err := s.mgr.Initialize(s.mockListener)
	s.NoError(err)

	s.Equal("license1", activeLicense.GetMetadata().GetId())
	s.Equal(v1.Metadata_VALID, s.mgr.GetLicenseStatus())

	license2 := &licenseproto.License{
		Metadata: &licenseproto.License_Metadata{
			Id: "license2",
		},
		Restrictions: &licenseproto.License_Restrictions{
			NotValidBefore:                     protoconv.ConvertTimeToTimestamp(time.Now()),
			NotValidAfter:                      protoconv.ConvertTimeToTimestamp(time.Now().Add(1 * time.Hour)),
			AllowOffline:                       true,
			NoNodeRestriction:                  true,
			NoBuildFlavorRestriction:           true,
			NoDeploymentEnvironmentRestriction: true,
		},
	}

	s.mockValidator.EXPECT().ValidateLicenseKey("KEY2").Return(license2, nil)
	s.mockStore.EXPECT().UpsertLicenseKeys(testutils.AssertionMatcher(assert.ElementsMatch, []*storage.StoredLicenseKey{
		{
			LicenseKey: "KEY2",
			LicenseId:  "license2",
			Selected:   true,
		},
		{
			LicenseKey: "KEY1",
			LicenseId:  "license1",
		},
	}))
	s.mockListener.EXPECT().OnActiveLicenseChanged(
		testutils.PredMatcher("new license is license 2 and is valid", func(l *v1.LicenseInfo) bool {
			if l.GetLicense().GetMetadata().GetId() != "license2" {
				return false
			}
			return l.GetStatus() == v1.LicenseInfo_VALID
		}),
		testutils.PredMatcher("old license is license 1 and is valid", func(l *v1.LicenseInfo) bool {
			if l.GetLicense().GetMetadata().GetId() != "license1" {
				return false
			}
			return l.GetStatus() == v1.LicenseInfo_VALID
		}),
	)

	addedLicense, err := s.mgr.AddLicenseKey("KEY2", true)
	s.NoError(err)
	s.Equal("license2", addedLicense.GetLicense().GetMetadata().GetId())
	s.Equal(v1.LicenseInfo_VALID, addedLicense.GetStatus())
	s.True(addedLicense.GetActive())

	activeLicense = s.mgr.GetActiveLicense()
	s.Equal("license2", activeLicense.GetMetadata().GetId())
	s.Equal(v1.Metadata_VALID, s.mgr.GetLicenseStatus())
}

func (s *managerTestSuite) TestLicenseActivatedAfterAdded() {
	s.mockStore.EXPECT().ListLicenseKeys().Return(nil, nil)

	s.mockValidator.EXPECT().ValidateLicenseKey("KEY1").Return(&licenseproto.License{
		Metadata: &licenseproto.License_Metadata{
			Id: "license1",
		},
		Restrictions: &licenseproto.License_Restrictions{
			NotValidBefore:                     protoconv.ConvertTimeToTimestamp(time.Now().Add(1 * time.Second)),
			NotValidAfter:                      protoconv.ConvertTimeToTimestamp(time.Now().Add(10 * time.Second)),
			AllowOffline:                       true,
			NoNodeRestriction:                  true,
			NoBuildFlavorRestriction:           true,
			NoDeploymentEnvironmentRestriction: true,
		},
	}, nil)

	s.mockListener.EXPECT().OnInitialize(s.mgr, gomock.Nil())

	activeLicense, err := s.mgr.Initialize(s.mockListener)
	s.NoError(err)
	s.Nil(activeLicense)
	s.Equal(v1.Metadata_NONE, s.mgr.GetLicenseStatus())

	s.mockStore.EXPECT().UpsertLicenseKeys([]*storage.StoredLicenseKey{
		{
			LicenseKey: "KEY1",
			LicenseId:  "license1",
			Selected:   false,
		},
	})

	addedLicense, err := s.mgr.AddLicenseKey("KEY1", false)
	s.Require().NoError(err)

	s.Equal("license1", addedLicense.GetLicense().GetMetadata().GetId())
	s.Equal(v1.LicenseInfo_NOT_YET_VALID, addedLicense.GetStatus())
	s.Equal(v1.Metadata_INVALID, s.mgr.GetLicenseStatus())

	newActiveLicense := s.mgr.GetActiveLicense()
	s.Nil(newActiveLicense)

	s.mockListener.EXPECT().OnActiveLicenseChanged(
		testutils.PredMatcher("new license is license 1 and is valid", func(l *v1.LicenseInfo) bool {
			if l.GetLicense().GetMetadata().GetId() != "license1" {
				return false
			}
			return l.GetStatus() == v1.LicenseInfo_VALID
		}),
		nil)

	s.mockStore.EXPECT().UpsertLicenseKeys([]*storage.StoredLicenseKey{
		{
			LicenseKey: "KEY1",
			LicenseId:  "license1",
			Selected:   true,
		},
	})

	time.Sleep(1500 * time.Millisecond)

	newActiveLicense = s.mgr.GetActiveLicense()
	s.Equal("license1", newActiveLicense.GetMetadata().GetId())
	s.Equal(v1.Metadata_VALID, s.mgr.GetLicenseStatus())
}

func (s *managerTestSuite) TestLicenseExpiredAfterAdded() {
	s.mockStore.EXPECT().ListLicenseKeys().Return(nil, nil)

	s.mockValidator.EXPECT().ValidateLicenseKey("KEY1").Return(&licenseproto.License{
		Metadata: &licenseproto.License_Metadata{
			Id: "license1",
		},
		Restrictions: &licenseproto.License_Restrictions{
			NotValidBefore:                     protoconv.ConvertTimeToTimestamp(time.Now().Add(-10 * time.Second)),
			NotValidAfter:                      protoconv.ConvertTimeToTimestamp(time.Now().Add(1 * time.Second)),
			AllowOffline:                       true,
			NoNodeRestriction:                  true,
			NoBuildFlavorRestriction:           true,
			NoDeploymentEnvironmentRestriction: true,
		},
	}, nil)

	s.mockListener.EXPECT().OnInitialize(s.mgr, gomock.Nil())

	activeLicense, err := s.mgr.Initialize(s.mockListener)
	s.NoError(err)
	s.Nil(activeLicense)
	s.Equal(v1.Metadata_NONE, s.mgr.GetLicenseStatus())

	s.mockListener.EXPECT().OnActiveLicenseChanged(
		testutils.PredMatcher("new license is license 1 and is valid", func(l *v1.LicenseInfo) bool {
			if l.GetLicense().GetMetadata().GetId() != "license1" {
				return false
			}
			return l.GetStatus() == v1.LicenseInfo_VALID
		}),
		nil)

	s.mockStore.EXPECT().UpsertLicenseKeys([]*storage.StoredLicenseKey{
		{
			LicenseKey: "KEY1",
			LicenseId:  "license1",
			Selected:   true,
		},
	})

	addedLicense, err := s.mgr.AddLicenseKey("KEY1", false)
	s.Require().NoError(err)

	s.Equal("license1", addedLicense.GetLicense().GetMetadata().GetId())
	s.Equal(v1.LicenseInfo_VALID, addedLicense.GetStatus())
	s.True(addedLicense.GetActive())

	newActiveLicense := s.mgr.GetActiveLicense()
	s.Equal(addedLicense.GetLicense(), newActiveLicense)
	s.Equal(v1.Metadata_VALID, s.mgr.GetLicenseStatus())

	s.mockListener.EXPECT().OnActiveLicenseChanged(
		nil,
		testutils.PredMatcher("old license is license 1 and is expired", func(l *v1.LicenseInfo) bool {
			if l.GetLicense().GetMetadata().GetId() != "license1" {
				return false
			}
			return l.GetStatus() == v1.LicenseInfo_EXPIRED
		}))

	s.mockStore.EXPECT().UpsertLicenseKeys([]*storage.StoredLicenseKey{
		{
			LicenseKey: "KEY1",
			LicenseId:  "license1",
			Selected:   false,
		},
	})

	time.Sleep(1500 * time.Millisecond)

	newActiveLicense = s.mgr.GetActiveLicense()
	s.Nil(newActiveLicense)
	s.Equal(v1.Metadata_EXPIRED, s.mgr.GetLicenseStatus())
}
