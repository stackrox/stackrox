//go:build sql_integration

package service

import (
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
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

func (s *serviceImplPermissionSetTestSuite) TestPostPermissionSet() {
	s.Run("Permission set without specified origin can be created by API", func() {
		inputPermissionSet := &storage.PermissionSet{
			Name: "Test basic permission set",
		}
		ctx := sac.WithAllAccess(s.T().Context())
		permissionSet, err := s.tester.service.PostPermissionSet(ctx, inputPermissionSet)
		s.NoError(err)
		inputPermissionSet.Id = permissionSet.GetId()
		protoassert.Equal(s.T(), inputPermissionSet, permissionSet)
	})
	s.Run("Dynamic scopes cannot be created by API", func() {
		inputScope := &storage.SimpleAccessScope{
			Traits: dynamicOriginTraits,
		}
		ctx := sac.WithAllAccess(s.T().Context())
		scope, err := s.tester.service.PostSimpleAccessScope(ctx, inputScope)
		s.ErrorIs(err, errox.InvalidArgs)
		s.Nil(scope)
	})
}

func (s *serviceImplPermissionSetTestSuite) TestPutPermissionSet() {
	s.Run("Permission set without specified origin can be updated by API", func() {
		permissionSetName := "Permission set without origin"
		inputPermissionSet := s.tester.createPermissionSet(s.T(), permissionSetName, nilTraits)
		updatedPermissionSet := inputPermissionSet.CloneVT()
		updatedPermissionSet.Description = "Updated description"
		ctx := sac.WithAllAccess(s.T().Context())
		_, err := s.tester.service.PutPermissionSet(ctx, updatedPermissionSet)
		s.NoError(err)
	})
	s.Run("Dynamic scopes cannot be created by API", func() {
		permissionSetName := "Dynamic permission set"
		inputPermissionSet := s.tester.createPermissionSet(s.T(), permissionSetName, dynamicOriginTraits)
		updatedPermissionSet := inputPermissionSet.CloneVT()
		updatedPermissionSet.Description = "Updated description"
		ctx := sac.WithAllAccess(s.T().Context())
		_, err := s.tester.service.PutPermissionSet(ctx, updatedPermissionSet)
		s.ErrorIs(err, errox.InvalidArgs)
	})
}
