//go:build sql_integration

package datastore

import (
	"context"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/sac/testconsts"
	"github.com/stackrox/rox/pkg/sac/testutils"
	searchPkg "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

func TestProcessBaselineDatastoreSAC(t *testing.T) {
	suite.Run(t, new(processBaselineSACTestSuite))
}

type processBaselineSACTestSuite struct {
	suite.Suite

	pool postgres.DB

	datastore DataStore

	optionsMap searchPkg.OptionsMap

	testContexts map[string]context.Context

	testProcessBaselineIDs []string
}

func (s *processBaselineSACTestSuite) SetupSuite() {
	pgtestbase := pgtest.ForT(s.T())
	s.Require().NotNil(pgtestbase)
	s.pool = pgtestbase.DB
	s.datastore = GetTestPostgresDataStore(s.T(), s.pool)
	s.optionsMap = schema.ProcessBaselinesSchema.OptionsMap

	s.testContexts = testutils.GetNamespaceScopedTestContexts(context.Background(), s.T(),
		resources.DeploymentExtension)
}

func (s *processBaselineSACTestSuite) TearDownSuite() {
	s.pool.Close()
}

func (s *processBaselineSACTestSuite) SetupTest() {
	s.testProcessBaselineIDs = make([]string, 0)

	processBaselines := fixtures.GetSACTestResourceSet(fixtures.GetScopedProcessBaseline)

	for i := range processBaselines {
		_, err := s.datastore.AddProcessBaseline(s.testContexts[testutils.UnrestrictedReadWriteCtx],
			processBaselines[i])
		s.Require().NoError(err)
	}

	for _, rb := range processBaselines {
		s.testProcessBaselineIDs = append(s.testProcessBaselineIDs, rb.GetId())
	}
}

func (s *processBaselineSACTestSuite) TearDownTest() {
	s.Require().NoError(s.datastore.RemoveProcessBaselinesByIDs(s.testContexts[testutils.UnrestrictedReadWriteCtx],
		s.testProcessBaselineIDs))
}

func (s *processBaselineSACTestSuite) deleteProcessBaseline(id string) {
	if id != "" {
		s.Require().NoError(s.datastore.RemoveProcessBaselinesByIDs(s.testContexts[testutils.UnrestrictedReadWriteCtx],
			[]string{id}))
	}
}

func (s *processBaselineSACTestSuite) TestAddProcessBaseline() {
	cases := testutils.GenericNamespaceSACUpsertTestCases(s.T(), testutils.VerbAdd)

	for name, c := range cases {
		s.Run(name, func() {
			processBaseline := fixtures.GetScopedProcessBaseline(uuid.NewV4().String(), testconsts.Cluster2,
				testconsts.NamespaceB)
			ctx := s.testContexts[c.ScopeKey]
			key, err := s.datastore.AddProcessBaseline(ctx, processBaseline)
			defer s.deleteProcessBaseline(key)
			if c.ExpectError {
				s.Require().Error(err)
				s.ErrorIs(err, c.ExpectedError)
			} else {
				s.NoError(err)
			}
		})
	}
}

func (s *processBaselineSACTestSuite) TestUpsertProcessBaseline() {
	cases := testutils.GenericNamespaceSACUpsertTestCases(s.T(), testutils.VerbUpsert)

	for name, c := range cases {
		s.Run(name, func() {
			processBaseline := fixtures.GetScopedProcessBaseline(uuid.NewV4().String(), testconsts.Cluster2,
				testconsts.NamespaceB)
			ctx := s.testContexts[c.ScopeKey]
			processBaseline, err := s.datastore.UpsertProcessBaseline(ctx, processBaseline.GetKey(), nil, false, false)
			defer s.deleteProcessBaseline(processBaseline.GetId())
			if c.ExpectError {
				s.Require().Error(err)
				s.ErrorIs(err, c.ExpectedError)
			} else {
				s.NoError(err)
			}
		})
	}
}

func (s *processBaselineSACTestSuite) TestUpdateProcessBaselineElements() {
	processBaseline := fixtures.GetScopedProcessBaseline(uuid.NewV4().String(), testconsts.Cluster2,
		testconsts.NamespaceB)
	_, err := s.datastore.AddProcessBaseline(s.testContexts[testutils.UnrestrictedReadWriteCtx], processBaseline)
	s.Require().NoError(err)
	s.testProcessBaselineIDs = append(s.testProcessBaselineIDs, processBaseline.GetId())

	cases := testutils.GenericNamespaceSACUpsertTestCases(s.T(), testutils.VerbUpdate)

	for name, c := range cases {
		s.Run(name, func() {
			ctx := s.testContexts[c.ScopeKey]
			_, err := s.datastore.UpdateProcessBaselineElements(
				ctx, processBaseline.GetKey(), nil, nil, false)
			if c.ExpectError {
				s.Require().Error(err)
				// if the requester does not have the necessary read permission,
				// the error will be "no process baseline with id XXX"
				// otherwise it will be "access to resource denied"
			} else {
				s.NoError(err)
			}
		})
	}
}

func (s *processBaselineSACTestSuite) TestGetProcessBaseline() {
	processBaseline := fixtures.GetScopedProcessBaseline(uuid.NewV4().String(), testconsts.Cluster2,
		testconsts.NamespaceB)
	_, err := s.datastore.AddProcessBaseline(s.testContexts[testutils.UnrestrictedReadWriteCtx], processBaseline)
	s.Require().NoError(err)
	s.testProcessBaselineIDs = append(s.testProcessBaselineIDs, processBaseline.GetId())

	cases := testutils.GenericNamespaceSACGetTestCases(s.T())

	for name, c := range cases {
		s.Run(name, func() {
			ctx := s.testContexts[c.ScopeKey]
			res, found, err := s.datastore.GetProcessBaseline(ctx, processBaseline.GetKey())
			s.Require().NoError(err)
			if c.ExpectedFound {
				s.Require().True(found)
				protoassert.Equal(s.T(), processBaseline, res)
			} else {
				s.False(found)
				s.Nil(res)
			}
		})
	}
}

func (s *processBaselineSACTestSuite) TestRemoveProcessBaseline() {
	cases := testutils.GenericNamespaceSACDeleteTestCases(s.T())

	for name, c := range cases {
		s.Run(name, func() {
			processBaseline := fixtures.GetScopedProcessBaseline(uuid.NewV4().String(), testconsts.Cluster2,
				testconsts.NamespaceB)
			_, err := s.datastore.AddProcessBaseline(s.testContexts[testutils.UnrestrictedReadWriteCtx], processBaseline)
			s.Require().NoError(err)
			s.testProcessBaselineIDs = append(s.testProcessBaselineIDs, processBaseline.GetId())
			defer s.deleteProcessBaseline(processBaseline.GetId())

			ctx := s.testContexts[c.ScopeKey]
			err = s.datastore.RemoveProcessBaseline(ctx, processBaseline.GetKey())
			s.NoError(err)

			fetched, found, err := s.datastore.GetProcessBaseline(
				s.testContexts[testutils.UnrestrictedReadWriteCtx],
				processBaseline.GetKey(),
			)
			s.NoError(err)
			if c.ExpectedFound {
				s.True(found)
				protoassert.Equal(s.T(), processBaseline, fetched)
			} else {
				s.False(found)
				s.Nil(fetched)
			}
		})
	}
}

func (s *processBaselineSACTestSuite) TestRemoveProcessBaselineByDeployment() {
	cases := testutils.GenericNamespaceSACDeleteTestCases(s.T())

	for name, c := range cases {
		s.Run(name, func() {
			processBaseline := fixtures.GetScopedProcessBaseline(uuid.NewV4().String(), testconsts.Cluster2,
				testconsts.NamespaceB)
			deploymentID := uuid.NewV4().String()
			processBaseline.Key.DeploymentId = deploymentID
			_, err := s.datastore.AddProcessBaseline(s.testContexts[testutils.UnrestrictedReadWriteCtx], processBaseline)
			s.Require().NoError(err)
			s.testProcessBaselineIDs = append(s.testProcessBaselineIDs, processBaseline.GetId())
			defer s.deleteProcessBaseline(processBaseline.GetId())

			ctx := s.testContexts[c.ScopeKey]
			err = s.datastore.RemoveProcessBaselinesByDeployment(ctx, deploymentID)
			s.NoError(err)

			fetched, found, err := s.datastore.GetProcessBaseline(
				s.testContexts[testutils.UnrestrictedReadWriteCtx],
				processBaseline.GetKey(),
			)
			s.NoError(err)
			if c.ExpectedFound {
				s.True(found)
				protoassert.Equal(s.T(), processBaseline, fetched)
			} else {
				s.False(found)
				s.Nil(fetched)
			}
		})
	}
}

func (s *processBaselineSACTestSuite) TestRemoveProcessBaselineByDeploymentOtherDeployment() {
	cases := testutils.GenericNamespaceSACDeleteTestCases(s.T())

	for name, c := range cases {
		s.Run(name, func() {
			processBaseline := fixtures.GetScopedProcessBaseline(uuid.NewV4().String(), testconsts.Cluster2,
				testconsts.NamespaceB)
			deploymentID := uuid.NewV4().String()
			otherDeploymentID := uuid.NewV4().String()
			s.Require().NotEqual(deploymentID, otherDeploymentID)
			processBaseline.Key.DeploymentId = deploymentID
			_, err := s.datastore.AddProcessBaseline(s.testContexts[testutils.UnrestrictedReadWriteCtx], processBaseline)
			s.Require().NoError(err)
			s.testProcessBaselineIDs = append(s.testProcessBaselineIDs, processBaseline.GetId())
			defer s.deleteProcessBaseline(processBaseline.GetId())

			ctx := s.testContexts[c.ScopeKey]
			err = s.datastore.RemoveProcessBaselinesByDeployment(ctx, otherDeploymentID)
			s.NoError(err)

			fetched, found, err := s.datastore.GetProcessBaseline(
				s.testContexts[testutils.UnrestrictedReadWriteCtx],
				processBaseline.GetKey(),
			)
			s.NoError(err)
			s.True(found)
			protoassert.Equal(s.T(), processBaseline, fetched)
		})
	}
}

func (s *processBaselineSACTestSuite) TestUserLockProcessBaselineLock() {
	cases := testutils.GenericNamespaceSACUpsertTestCases(s.T(), "lock")

	for name, c := range cases {
		s.Run(name, func() {
			processBaseline := fixtures.GetScopedProcessBaseline(uuid.NewV4().String(), testconsts.Cluster2,
				testconsts.NamespaceB)
			id, err := keyToID(processBaseline.GetKey())
			s.Require().NoError(err)
			processBaseline.Id = id
			ctx := s.testContexts[c.ScopeKey]
			_, err = s.datastore.AddProcessBaseline(
				s.testContexts[testutils.UnrestrictedReadWriteCtx],
				processBaseline,
			)
			defer s.deleteProcessBaseline(processBaseline.GetId())
			expectedUnchanged := processBaseline.CloneVT()

			_, err = s.datastore.UserLockProcessBaseline(ctx, processBaseline.GetKey(), true)
			fetched, found, fetchErr := s.datastore.GetProcessBaseline(
				s.testContexts[testutils.UnrestrictedReadWriteCtx],
				processBaseline.GetKey(),
			)
			s.NoError(fetchErr)
			s.True(found)
			if c.ExpectError {
				s.Require().Error(err)
				// if the requester does not have the necessary read permission,
				// the error will be "no process baseline with id XXX"
				// otherwise it will be "access to resource denied"

				// Ensure the process baseline was not changed
				protoassert.Equal(s.T(), expectedUnchanged, fetched)
			} else {
				s.NoError(err)

				// Ensure the user lock timestamp was changed
				assert.NotNil(s.T(), fetched.GetUserLockedTimestamp())
				assert.Nil(s.T(), expectedUnchanged.GetUserLockedTimestamp())
				// Ensure the last updated state was changed
				assert.NotEqual(s.T(), expectedUnchanged.GetLastUpdate().GetNanos(), fetched.GetLastUpdate().GetNanos())
				// Ensure the above two changes were the only changes
				expectedUnchanged.LastUpdate = nil
				expectedUnchanged.UserLockedTimestamp = nil
				fetched.LastUpdate = nil
				fetched.UserLockedTimestamp = nil
				protoassert.Equal(s.T(), expectedUnchanged, fetched)
			}
		})
	}
}

func (s *processBaselineSACTestSuite) TestUserLockProcessBaselineUnlock() {
	cases := testutils.GenericNamespaceSACUpsertTestCases(s.T(), "unlock")

	for name, c := range cases {
		s.Run(name, func() {
			processBaseline := fixtures.GetScopedProcessBaseline(uuid.NewV4().String(), testconsts.Cluster2,
				testconsts.NamespaceB)
			id, err := keyToID(processBaseline.GetKey())
			s.Require().NoError(err)
			processBaseline.Id = id
			userLockTS := time.Date(2022, 3, 4, 5, 6, 0, 0, time.UTC)
			userLockProtoTS := protocompat.ConvertTimeToTimestampOrNil(&userLockTS)
			s.Require().NotNil(userLockProtoTS)
			processBaseline.UserLockedTimestamp = userLockProtoTS
			ctx := s.testContexts[c.ScopeKey]
			_, err = s.datastore.AddProcessBaseline(
				s.testContexts[testutils.UnrestrictedReadWriteCtx],
				processBaseline,
			)
			defer s.deleteProcessBaseline(processBaseline.GetId())
			expectedUnchanged := processBaseline.CloneVT()

			_, err = s.datastore.UserLockProcessBaseline(ctx, processBaseline.GetKey(), false)
			fetched, found, fetchErr := s.datastore.GetProcessBaseline(
				s.testContexts[testutils.UnrestrictedReadWriteCtx],
				processBaseline.GetKey(),
			)
			s.NoError(fetchErr)
			s.True(found)
			if c.ExpectError {
				s.Require().Error(err)
				// if the requester does not have the necessary read permission,
				// the error will be "no process baseline with id XXX"
				// otherwise it will be "access to resource denied"

				// Ensure the process baseline was not changed
				protoassert.Equal(s.T(), expectedUnchanged, fetched)
			} else {
				s.NoError(err)

				// Ensure the user lock timestamp was changed
				assert.Nil(s.T(), fetched.GetUserLockedTimestamp())
				assert.NotNil(s.T(), expectedUnchanged.GetUserLockedTimestamp())
				// Ensure the last updated state was changed
				assert.NotEqual(s.T(), expectedUnchanged.GetLastUpdate().GetNanos(), fetched.GetLastUpdate().GetNanos())
				// Ensure the above two changes were the only changes
				expectedUnchanged.LastUpdate = nil
				expectedUnchanged.UserLockedTimestamp = nil
				fetched.LastUpdate = nil
				fetched.UserLockedTimestamp = nil
				protoassert.Equal(s.T(), expectedUnchanged, fetched)
			}
		})
	}
}

func (s *processBaselineSACTestSuite) TestCreateUnlockedProcessBaseline() {
	cases := testutils.GenericNamespaceSACUpsertTestCases(s.T(), "create")

	for name, c := range cases {
		s.Run(name, func() {
			processBaseline := fixtures.GetScopedProcessBaseline(uuid.NewV4().String(), testconsts.Cluster2,
				testconsts.NamespaceB)
			id, err := keyToID(processBaseline.GetKey())
			s.Require().NoError(err)
			processBaseline.Id = id
			processBaseline.Elements = []*storage.BaselineElement{}
			ctx := s.testContexts[c.ScopeKey]
			created, err := s.datastore.CreateUnlockedProcessBaseline(ctx, processBaseline.GetKey())
			defer s.deleteProcessBaseline(processBaseline.GetId())

			fetched, found, fetchErr := s.datastore.GetProcessBaseline(
				s.testContexts[testutils.UnrestrictedReadWriteCtx],
				processBaseline.GetKey(),
			)
			s.NoError(fetchErr)
			if c.ExpectError {
				s.Require().Error(err)
				s.ErrorIs(err, c.ExpectedError)

				s.False(found)
				s.Nil(fetched)
			} else {
				s.NoError(err)

				s.True(found)
				s.NotNil(fetched.GetCreated())
				s.NotNil(fetched.GetLastUpdate())
				s.NotNil(fetched.GetStackRoxLockedTimestamp())
				// Check the created baseline is empty
				protoassert.Equal(s.T(), created, fetched)
				fetched.Created = nil
				fetched.LastUpdate = nil
				fetched.StackRoxLockedTimestamp = nil
				processBaseline.Created = nil
				processBaseline.LastUpdate = nil
				processBaseline.StackRoxLockedTimestamp = nil
				protoassert.Equal(s.T(), processBaseline, fetched)
			}
		})
	}
}

func (s *processBaselineSACTestSuite) TestRemoveProcessBaselinesByID() {
	cases := testutils.GenericNamespaceSACDeleteTestCases(s.T())

	for name, c := range cases {
		s.Run(name, func() {
			processBaseline := fixtures.GetScopedProcessBaseline(uuid.NewV4().String(), testconsts.Cluster2,
				testconsts.NamespaceB)
			id, err := keyToID(processBaseline.GetKey())
			s.Require().NoError(err)
			processBaseline.Id = id
			ctx := s.testContexts[c.ScopeKey]
			_, err = s.datastore.AddProcessBaseline(
				s.testContexts[testutils.UnrestrictedReadWriteCtx],
				processBaseline,
			)
			s.Require().NoError(err)
			defer s.deleteProcessBaseline(processBaseline.GetId())

			err = s.datastore.RemoveProcessBaselinesByIDs(ctx, []string{id})
			s.NoError(err)
			fetched, found, fetchErr := s.datastore.GetProcessBaseline(
				s.testContexts[testutils.UnrestrictedReadWriteCtx],
				processBaseline.GetKey(),
			)
			s.NoError(fetchErr)
			if c.ExpectedFound {
				s.True(found)
				protoassert.Equal(s.T(), processBaseline, fetched)
			} else {
				s.False(found)
				s.Nil(fetched)
			}
		})
	}
}

func (s *processBaselineSACTestSuite) TestClearProcessBaselines() {
	cases := testutils.GenericNamespaceSACUpsertTestCases(s.T(), "clear")

	for name, c := range cases {
		s.Run(name, func() {
			processBaseline := fixtures.GetScopedProcessBaseline(uuid.NewV4().String(), testconsts.Cluster2,
				testconsts.NamespaceB)
			id, err := keyToID(processBaseline.GetKey())
			s.Require().NoError(err)
			processBaseline.Id = id
			processBaseline.Elements = []*storage.BaselineElement{}
			processBaseline.ElementGraveyard = []*storage.BaselineElement{}
			ctx := s.testContexts[c.ScopeKey]
			_, err = s.datastore.AddProcessBaseline(s.testContexts[testutils.UnrestrictedReadWriteCtx], processBaseline)
			s.Require().NoError(err)
			defer s.deleteProcessBaseline(processBaseline.GetId())

			err = s.datastore.ClearProcessBaselines(ctx, []string{id})
			fetched, found, fetchErr := s.datastore.GetProcessBaseline(
				s.testContexts[testutils.UnrestrictedReadWriteCtx],
				processBaseline.GetKey(),
			)
			s.NoError(fetchErr)
			s.True(found)
			if c.ExpectError {
				// Depending whether the requester has read access to the target object or not,
				// the error behaviour of ClearProcessBaselines will be different
				// - If read is granted, the user will get an `access to resource denied` error
				// - otherwise, there will be no error
				if sac.ForResource(resources.DeploymentExtension).
					ScopeChecker(ctx, storage.Access_READ_ACCESS).
					ForNamespaceScopedObject(processBaseline.GetKey()).IsAllowed() {
					s.Error(err)
					s.ErrorIs(err, sac.ErrResourceAccessDenied)
				} else {
					s.NoError(err)
				}

				// Ensure the baseline was not changed
				protoassert.Equal(s.T(), processBaseline, fetched)
			} else {
				s.NoError(err)

				// Ensure the following fields were changed
				// - LastUpdate
				// - StackRoxLockedTimestamp
				// - Elements
				// - ElementGraveyard
				s.NotNil(fetched.GetLastUpdate())
				s.NotEqual(processBaseline.GetLastUpdate().GetNanos(), fetched.GetLastUpdate().GetNanos())
				s.NotNil(fetched.GetStackRoxLockedTimestamp())
				s.NotEqual(processBaseline.GetStackRoxLockedTimestamp().GetNanos(), fetched.GetStackRoxLockedTimestamp().GetNanos())
				s.Nil(fetched.GetElements())
				s.NotNil(processBaseline.GetElements())
				s.Nil(fetched.GetElementGraveyard())
				s.NotNil(processBaseline.GetElementGraveyard())
				// Ensure these changes were the only changes
				processBaseline.LastUpdate = nil
				fetched.LastUpdate = nil
				processBaseline.StackRoxLockedTimestamp = nil
				fetched.StackRoxLockedTimestamp = nil
				processBaseline.Elements = nil
				processBaseline.ElementGraveyard = nil
				protoassert.Equal(s.T(), processBaseline, fetched)

			}
		})
	}
}

func (s *processBaselineSACTestSuite) runSearchRawTest(c testutils.SACSearchTestCase) {
	ctx := s.testContexts[c.ScopeKey]
	results, err := s.datastore.SearchRawProcessBaselines(ctx, nil)
	s.Require().NoError(err)
	resultObjs := make([]sac.NamespaceScopedObject, 0, len(results))
	for i := range results {
		resultObjs = append(resultObjs, results[i].GetKey())
	}
	resultCounts := testutils.CountSearchResultObjectsPerClusterAndNamespace(s.T(), resultObjs)
	testutils.ValidateSACSearchResultDistribution(&s.Suite, c.Results, resultCounts)
}

func (s *processBaselineSACTestSuite) runSearchTest(c testutils.SACSearchTestCase) {
	ctx := s.testContexts[c.ScopeKey]
	results, err := s.datastore.Search(ctx, nil)
	s.Require().NoError(err)
	resultObjects := make([]sac.NamespaceScopedObject, 0, len(results))
	for _, r := range results {
		key, err := IDToKey(r.ID)
		if err != nil {
			continue
		}
		resultObjects = append(resultObjects, key)
	}
	resultCounts := testutils.CountSearchResultObjectsPerClusterAndNamespace(s.T(), resultObjects)
	testutils.ValidateSACSearchResultDistribution(&s.Suite, c.Results, resultCounts)
}

func (s *processBaselineSACTestSuite) TestScopedSearch() {
	for name, c := range testutils.GenericScopedSACSearchTestCases(s.T()) {
		s.Run(name, func() {
			s.runSearchTest(c)
		})
	}
}

func (s *processBaselineSACTestSuite) TestUnrestrictedSearch() {
	for name, c := range testutils.GenericUnrestrictedRawSACSearchTestCases(s.T()) {
		s.Run(name, func() {
			s.runSearchTest(c)
		})
	}
}

func (s *processBaselineSACTestSuite) TestScopeSearchRaw() {
	for name, c := range testutils.GenericScopedSACSearchTestCases(s.T()) {
		s.Run(name, func() {
			s.runSearchRawTest(c)
		})
	}
}

func (s *processBaselineSACTestSuite) TestUnrestrictedSearchRaw() {
	for name, c := range testutils.GenericUnrestrictedRawSACSearchTestCases(s.T()) {
		s.Run(name, func() {
			s.runSearchRawTest(c)
		})
	}
}
