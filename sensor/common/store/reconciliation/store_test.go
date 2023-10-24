package reconciliation

import (
	"testing"

	"github.com/stackrox/rox/pkg/set"
	"github.com/stretchr/testify/suite"
)

const (
	resourceTypeA = "resource_type_A"
	resourceTypeB = "resource_type_B"
	fixtureID1    = "1"
	fixtureID2    = "2"
)

type storeSuite struct {
	suite.Suite
	resourceStore *store
}

var _ suite.SetupTestSuite = (*storeSuite)(nil)

func (s *storeSuite) SetupTest() {
	var ok bool
	s.resourceStore, ok = NewStore().(*store)
	s.Assert().True(ok)
}

func Test_StoreSuite(t *testing.T) {
	suite.Run(t, new(storeSuite))
}

func (s *storeSuite) Test_UpsertType() {
	testCases := map[string]struct {
		inputTypes    []string
		expectedTypes []string
	}{
		"One resource": {
			inputTypes:    []string{resourceTypeA},
			expectedTypes: []string{resourceTypeA},
		},
		"Same resource is only added once": {
			inputTypes:    []string{resourceTypeA, resourceTypeA},
			expectedTypes: []string{resourceTypeA},
		},
		"Multiple resource types": {
			inputTypes:    []string{resourceTypeA, resourceTypeB},
			expectedTypes: []string{resourceTypeA, resourceTypeB},
		},
	}
	for name, tc := range testCases {
		s.Run(name, func() {
			s.resourceStore.resourceTypes = set.NewStringSet()
			for _, input := range tc.inputTypes {
				s.resourceStore.UpsertType(input)
			}
			s.Assert().Len(s.resourceStore.resourceTypes, len(tc.expectedTypes))
			for _, resType := range tc.expectedTypes {
				_, found := s.resourceStore.resources[resType]
				s.Require().True(found)
				s.Assert().Contains(s.resourceStore.resourceTypes, resType)
			}
		})
	}
}

func (s *storeSuite) Test_Upsert() {
	testCases := map[string]struct {
		inputResources    map[string][]string
		expectedResources map[string][]string
	}{
		"One resource": {
			inputResources: map[string][]string{
				resourceTypeA: {fixtureID1},
			},
			expectedResources: map[string][]string{
				resourceTypeA: {fixtureID1},
			},
		},
		"Same resource is only added once": {
			inputResources: map[string][]string{
				resourceTypeA: {fixtureID1, fixtureID1},
			},
			expectedResources: map[string][]string{
				resourceTypeA: {fixtureID1},
			},
		},
		"Multiple resource types": {
			inputResources: map[string][]string{
				resourceTypeA: {fixtureID1, fixtureID2},
				resourceTypeB: {fixtureID1},
			},
			expectedResources: map[string][]string{
				resourceTypeA: {fixtureID1, fixtureID2},
				resourceTypeB: {fixtureID1},
			},
		},
	}
	for name, tc := range testCases {
		s.Run(name, func() {
			s.resourceStore.resources = make(map[string]set.StringSet)
			for inputType, input := range tc.inputResources {
				for _, res := range input {
					s.resourceStore.Upsert(inputType, res)
				}
			}
			for resType, expectedResources := range tc.expectedResources {
				resources, found := s.resourceStore.resources[resType]
				s.Require().True(found)
				s.Assert().Len(expectedResources, len(resources))
				for _, resource := range expectedResources {
					s.Assert().Contains(resources, resource)
				}
			}
		})
	}
}

func (s *storeSuite) initializeStore() {
	s.resourceStore.Cleanup()
	s.resourceStore.Upsert(resourceTypeA, fixtureID1)
	s.resourceStore.Upsert(resourceTypeA, fixtureID2)
	s.resourceStore.Upsert(resourceTypeB, fixtureID1)
	s.resourceStore.Upsert(resourceTypeB, fixtureID2)
}

func (s *storeSuite) Test_Remove() {
	testCases := map[string]struct {
		resourcesToRemove map[string][]string
		expectedResources map[string][]string
	}{
		"Remove one": {
			resourcesToRemove: map[string][]string{
				resourceTypeA: {fixtureID1},
			},
			expectedResources: map[string][]string{
				resourceTypeA: {fixtureID2},
				resourceTypeB: {fixtureID1, fixtureID2},
			},
		},
		"Remove twice": {
			resourcesToRemove: map[string][]string{
				resourceTypeA: {fixtureID1, fixtureID1},
			},
			expectedResources: map[string][]string{
				resourceTypeA: {fixtureID2},
				resourceTypeB: {fixtureID1, fixtureID2},
			},
		},
		"Remove with incorrect id": {
			resourcesToRemove: map[string][]string{
				resourceTypeA: {"incorrect id"},
			},
			expectedResources: map[string][]string{
				resourceTypeA: {fixtureID1, fixtureID2},
				resourceTypeB: {fixtureID1, fixtureID2},
			},
		},
		"Remove with incorrect type": {
			resourcesToRemove: map[string][]string{
				"incorrect type": {fixtureID1},
			},
			expectedResources: map[string][]string{
				resourceTypeA: {fixtureID1, fixtureID2},
				resourceTypeB: {fixtureID1, fixtureID2},
			},
		},
		"Remove all": {
			resourcesToRemove: map[string][]string{
				resourceTypeA: {fixtureID1, fixtureID2},
				resourceTypeB: {fixtureID1, fixtureID2},
			},
			expectedResources: map[string][]string{},
		},
	}
	for name, tc := range testCases {
		s.Run(name, func() {
			s.initializeStore()
			for inputType, input := range tc.resourcesToRemove {
				for _, res := range input {
					s.resourceStore.Remove(inputType, res)
				}
			}
			for resType, expectedResources := range tc.expectedResources {
				resources, found := s.resourceStore.resources[resType]
				s.Require().True(found)
				s.Assert().Len(expectedResources, len(resources))
				for _, resource := range expectedResources {
					s.Assert().Contains(resources, resource)
				}
			}
		})
	}
}

func (s *storeSuite) Test_Cleanup() {
	s.resourceStore.Upsert(resourceTypeA, fixtureID1)
	s.resourceStore.Upsert(resourceTypeA, fixtureID2)
	s.resourceStore.Upsert(resourceTypeB, fixtureID1)
	s.resourceStore.Upsert(resourceTypeB, fixtureID2)

	s.resourceStore.Cleanup()

	for _, ids := range s.resourceStore.resources {
		s.Assert().Len(ids, 0)
	}
	s.Assert().Equal(len(s.resourceStore.resourceTypes), len(s.resourceStore.resources))
	s.Assert().Equal(2, len(s.resourceStore.resourceTypes))
}
