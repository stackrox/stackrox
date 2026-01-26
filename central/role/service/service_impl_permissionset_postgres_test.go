package service

import (
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/suite"
)

func TestServiceImplWithDB_PermissionSet(t *testing.T) {
	suite.Run(t, new(serviceImplPermissionSetTestSuite))
}

type serviceImplPermissionSetTestSuite struct {
	suite.Suite

	tester *serviceImplTester
}

func (s *serviceImplPermissionSetTestSuite) SetupSuite() {
	s.tester = &serviceImplTester{}
	s.tester.Setup(s.T())
}

func (s *serviceImplPermissionSetTestSuite) SetupTest() {
	s.Require().NotNil(s.tester)
	s.tester.SetupTest(s.T())
}

func (s *serviceImplPermissionSetTestSuite) TearDownTest() {
	s.Require().NotNil(s.tester)
	s.tester.TearDownTest(s.T())
}

func (s *serviceImplPermissionSetTestSuite) TestListPermissionSets() {
	t := s.T()
	ctx := sac.WithAllAccess(t.Context())

	permissionSetName1 := "TestListPermissionSets_noTraits"
	permissionSetName2 := "TestListPermissionSets_imperativeOriginTraits"
	permissionSetName3 := "TestListPermissionSets_declarativeOriginTraits"
	permissionSetName4 := "TestListPermissionSets_orphanedDeclarativeOriginTraits"
	permissionSetName5 := "TestListPermissionSets_dynamicOriginTraits"
	permissionSet1 := s.tester.createPermissionSet(t, permissionSetName1, nilTraits)
	permissionSet2 := s.tester.createPermissionSet(t, permissionSetName2, imperativeOriginTraits)
	permissionSet3 := s.tester.createPermissionSet(t, permissionSetName3, declarativeOriginTraits)
	permissionSet4 := s.tester.createPermissionSet(t, permissionSetName4, orphanedDeclarativeOriginTraits)
	permissionSet5 := s.tester.createPermissionSet(t, permissionSetName5, dynamicOriginTraits)

	permissionSets, err := s.tester.service.ListPermissionSets(ctx, &v1.Empty{})
	s.NoError(err)
	s.Len(permissionSets.GetPermissionSets(), 4)

	protoassert.SliceContains(s.T(), permissionSets.GetPermissionSets(), permissionSet1)
	protoassert.SliceContains(s.T(), permissionSets.GetPermissionSets(), permissionSet2)
	protoassert.SliceContains(s.T(), permissionSets.GetPermissionSets(), permissionSet3)
	protoassert.SliceContains(s.T(), permissionSets.GetPermissionSets(), permissionSet4)
	// Roles with dynamic origin are filtered out.
	protoassert.SliceNotContains(s.T(), permissionSets.GetPermissionSets(), permissionSet5)
}
