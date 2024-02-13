//go:build sql_integration

package postgres

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	pkgSchema "github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/postgres/walker"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/sac/testconsts"
	"github.com/stackrox/rox/pkg/sac/testutils"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

var (
	ctx = sac.WithAllAccess(context.Background())
)

func TestNewStore(t *testing.T) {
	testDB := pgtest.ForT(t)
	assert.NotNil(t, newStore(testDB))
}

func TestNewGenericStore(t *testing.T) {
	testDB := pgtest.ForT(t)
	assert.NotNil(t, NewGenericStore[storage.TestSingleKeyStruct, *storage.TestSingleKeyStruct](
		testDB.DB,
		pkgSchema.TestSingleKeyStructsSchema,
		pkGetter,
		insertIntoTestSingleKeyStructs,
		copyFromTestSingleKeyStructs,
		doNothingDurationTimeSetter,
		doNothingDurationTimeSetter,
		GloballyScopedUpsertChecker[storage.TestSingleKeyStruct, *storage.TestSingleKeyStruct](resources.Namespace),
		resources.Namespace,
	))
}

func TestUpsert(t *testing.T) {
	testDB := pgtest.ForT(t)
	store := newStore(testDB)
	key := "TestUpsert"
	name := "Test Upsert"
	testObject := newTestSingleKeyStruct(key, name, int64(1))

	objBefore, foundBefore, errBefore := store.Get(ctx, key)
	assert.Nil(t, objBefore)
	assert.False(t, foundBefore)
	assert.NoError(t, errBefore)

	assert.NoError(t, store.Upsert(ctx, testObject))

	objAfter, foundAfter, errAfter := store.Get(ctx, key)
	assert.Equal(t, testObject, objAfter)
	assert.True(t, foundAfter)
	assert.NoError(t, errAfter)
}

func TestUpsertMany(t *testing.T) {
	testDB := pgtest.ForT(t)
	store := newStore(testDB)

	testObjects := sampleTestSingleKeyStructArray("UpsertMany")

	for _, obj := range testObjects {
		objBefore, foundBefore, errBefore := store.Get(ctx, pkGetter(obj))
		assert.Nil(t, objBefore)
		assert.False(t, foundBefore)
		assert.NoError(t, errBefore)
	}

	assert.NoError(t, store.UpsertMany(ctx, testObjects))

	for _, obj := range testObjects {
		objAfter, foundAfter, errAfter := store.Get(ctx, pkGetter(obj))
		assert.Equal(t, obj, objAfter)
		assert.True(t, foundAfter)
		assert.NoError(t, errAfter)
	}
}

func TestDelete(t *testing.T) {
	testDB := pgtest.ForT(t)
	store := newStore(testDB)
	require.NotNil(t, store)

	key := "TestDelete"
	name := "Test Delete"
	testObject := newTestSingleKeyStruct(key, name, int64(1))
	require.NoError(t, store.Upsert(ctx, testObject))

	objBefore, foundBefore, errBefore := store.Get(ctx, key)
	require.Equal(t, testObject, objBefore)
	require.True(t, foundBefore)
	require.NoError(t, errBefore)

	assert.NoError(t, store.Delete(ctx, key))

	objAfter, foundAfter, errAfter := store.Get(ctx, key)
	require.Nil(t, objAfter)
	require.False(t, foundAfter)
	require.NoError(t, errAfter)

	missingKey := "TestDeleteMissingKey"

	missingObjBefore, missingFoundBefore, missingErrBefore := store.Get(ctx, missingKey)
	require.Nil(t, missingObjBefore)
	require.False(t, missingFoundBefore)
	require.NoError(t, missingErrBefore)

	assert.NoError(t, store.Delete(ctx, missingKey))

	missingObjAfter, missingFoundAfter, missingErrAfter := store.Get(ctx, missingKey)
	require.Nil(t, missingObjAfter)
	require.False(t, missingFoundAfter)
	require.NoError(t, missingErrAfter)

}

func TestDeleteMany(t *testing.T) {
	testDB := pgtest.ForT(t)
	store := newStore(testDB)
	require.NotNil(t, store)

	objectBatch := sampleTestSingleKeyStructArray("DeleteMany")
	require.NoError(t, store.UpsertMany(ctx, objectBatch))

	identifiersToRemove := make([]string, 0, len(objectBatch)+1)
	for _, obj := range objectBatch {
		key := pkGetter(obj)
		identifiersToRemove = append(identifiersToRemove, key)
		// ensure object is in DB before call to remove
		objBefore, foundBefore, errBefore := store.Get(ctx, key)
		assert.Equal(t, obj, objBefore)
		assert.True(t, foundBefore)
		assert.NoError(t, errBefore)
	}

	missingKey := "TestDeleteManyMissingKey"
	identifiersToRemove = append(identifiersToRemove, missingKey)
	missingObjBefore, missingFoundBefore, missingErrBefore := store.Get(ctx, missingKey)
	assert.Nil(t, missingObjBefore)
	assert.False(t, missingFoundBefore)
	assert.NoError(t, missingErrBefore)

	assert.NoError(t, store.DeleteMany(ctx, identifiersToRemove))

	for _, obj := range objectBatch {
		key := pkGetter(obj)
		// ensure object is NOT in DB after call to remove
		objAfter, foundAfter, errAfter := store.Get(ctx, key)
		assert.Nil(t, objAfter)
		assert.False(t, foundAfter)
		assert.NoError(t, errAfter)
	}

	missingObjAfter, missingFoundAfter, missingErrAfter := store.Get(ctx, missingKey)
	assert.Nil(t, missingObjAfter)
	assert.False(t, missingFoundAfter)
	assert.NoError(t, missingErrAfter)
}

func TestExists(t *testing.T) {
	testDB := pgtest.ForT(t)
	store := newStore(testDB)
	require.NotNil(t, store)

	key := "TestExists"
	name := "Test Exists"
	testObject := newTestSingleKeyStruct(key, name, int64(9))

	require.NoError(t, store.Upsert(ctx, testObject))

	missingKey := "TestExistsMissingKey"

	foundExisting, errExisting := store.Exists(ctx, key)
	assert.True(t, foundExisting)
	assert.NoError(t, errExisting)

	foundMissing, errMissing := store.Exists(ctx, missingKey)
	assert.False(t, foundMissing)
	assert.NoError(t, errMissing)
}

func TestCount(t *testing.T) {
	testDB := pgtest.ForT(t)
	store := newStore(testDB)
	require.NotNil(t, store)

	firstCount, err1 := store.Count(ctx)
	assert.Equal(t, 0, firstCount)
	assert.NoError(t, err1)

	testObject1 := newTestSingleKeyStruct("TestCount", "Test Count", int64(256))
	assert.NoError(t, store.Upsert(ctx, testObject1))

	secondCount, err2 := store.Count(ctx)
	assert.Equal(t, 1, secondCount)
	assert.NoError(t, err2)

	supplementaryObjects := sampleTestSingleKeyStructArray("Count")
	assert.NoError(t, store.UpsertMany(ctx, supplementaryObjects))

	thirdCount, err3 := store.Count(ctx)
	assert.Equal(t, 1+len(supplementaryObjects), thirdCount)
	assert.NoError(t, err3)
}

func TestWalk(t *testing.T) {
	testDB := pgtest.ForT(t)
	store := newStore(testDB)
	require.NotNil(t, store)

	testObjects := sampleTestSingleKeyStructArray("Walk")
	assert.NoError(t, store.UpsertMany(ctx, testObjects))

	injectedNames := make([]string, 0, len(testObjects))
	for _, obj := range testObjects {
		injectedNames = append(injectedNames, obj.GetName())
	}

	walkedNames := make([]string, 0, len(testObjects))
	walkedObjects := make([]*storage.TestSingleKeyStruct, 0, len(testObjects))

	walkFn := func(obj *storage.TestSingleKeyStruct) error {
		walkedNames = append(walkedNames, obj.Name)
		walkedObjects = append(walkedObjects, obj)
		return nil
	}

	assert.NoError(t, store.Walk(ctx, walkFn))

	assert.ElementsMatch(t, walkedNames, injectedNames)
	assert.ElementsMatch(t, testObjects, walkedObjects)
}

func TestWalkByQuery(t *testing.T) {
	testDB := pgtest.ForT(t)
	store := newStore(testDB)
	require.NotNil(t, store)

	testObjects := sampleTestSingleKeyStructArray("WalkByQuery")
	assert.NoError(t, store.UpsertMany(ctx, testObjects))

	query := search.NewQueryBuilder().AddExactMatches(search.TestName, testObjects[0].GetName(), testObjects[1].GetName()).ProtoQuery()
	expectedObjects := testObjects[:2]
	var walkedObjects []*storage.TestSingleKeyStruct
	walkFn := func(obj *storage.TestSingleKeyStruct) error {
		walkedObjects = append(walkedObjects, obj)
		return nil
	}

	assert.NoError(t, store.WalkByQuery(ctx, query, walkFn))
	assert.ElementsMatch(t, expectedObjects, walkedObjects)
}

func TestGetAll(t *testing.T) {
	testDB := pgtest.ForT(t)
	store := newStore(testDB)
	require.NotNil(t, store)

	testObjects := sampleTestSingleKeyStructArray("GetAll")
	assert.NoError(t, store.UpsertMany(ctx, testObjects))

	fetchedObjects, err := store.GetAll(ctx)
	assert.NoError(t, err)
	assert.ElementsMatch(t, fetchedObjects, testObjects)
}

func TestGetIDs(t *testing.T) {
	testDB := pgtest.ForT(t)
	store := newStore(testDB)
	require.NotNil(t, store)

	testObjects := sampleTestSingleKeyStructArray("GetIDs")
	assert.NoError(t, store.UpsertMany(ctx, testObjects))

	expectedIDs := make([]string, 0, len(testObjects))
	for _, obj := range testObjects {
		expectedIDs = append(expectedIDs, pkGetter(obj))
	}

	fetchedIDs, err := store.GetIDs(ctx)
	assert.NoError(t, err)
	assert.ElementsMatch(t, fetchedIDs, expectedIDs)
}

func TestGet(t *testing.T) {
	testDB := pgtest.ForT(t)
	store := newStore(testDB)
	key := "TestGet"
	name := "Test Get"
	testObject := newTestSingleKeyStruct(key, name, int64(15))

	missingKey := "TestGetMissing"

	assert.NoError(t, store.Upsert(ctx, testObject))

	// Object with ID "TestGet" is in DB
	obj, found, err := store.Get(ctx, key)
	assert.Equal(t, testObject, obj)
	assert.True(t, found)
	assert.NoError(t, err)

	// Object with ID "TestGetMissing" is NOT in DB
	missingObj, missingFound, missingErr := store.Get(ctx, missingKey)
	assert.Nil(t, missingObj)
	assert.False(t, missingFound)
	assert.NoError(t, missingErr)
}

func TestGetMany(t *testing.T) {
	testDB := pgtest.ForT(t)
	store := newStore(testDB)

	missingKey := "TestGetManyMissing"
	testObjects := sampleTestSingleKeyStructArray("GetMany")
	assert.NoError(t, store.UpsertMany(ctx, testObjects))

	identifiersToFetch := make([]string, 0, len(testObjects)+1)
	expectedObjects := make([]*storage.TestSingleKeyStruct, 0, len(testObjects))
	identifiersToFetch = append(identifiersToFetch, missingKey)
	for ix, obj := range testObjects {
		if ix%2 == 1 {
			continue
		}
		identifiersToFetch = append(identifiersToFetch, pkGetter(obj))
		expectedObjects = append(expectedObjects, obj)
	}

	fetchedObjects, missingIndices, err := store.GetMany(ctx, identifiersToFetch)
	assert.NoError(t, err)
	assert.ElementsMatch(t, fetchedObjects, expectedObjects)
	assert.Equal(t, []int{0}, missingIndices)
}

func TestGetByQuery(t *testing.T) {
	testDB := pgtest.ForT(t)
	store := newStore(testDB)

	testObjects := sampleTestSingleKeyStructArray("GetByQuery")
	query2 := getMatchFieldQuery("Test Name", "Test GetByQuery 2")
	query4 := getMatchFieldQuery("Test Key", "TestGetByQuery4")
	query := getDisjunctionQuery(query2, query4)

	objectsBefore, errBefore := store.GetByQuery(ctx, query)
	assert.NoError(t, errBefore)
	assert.Empty(t, objectsBefore)

	assert.NoError(t, store.UpsertMany(ctx, testObjects))

	objectsAfter, errAfter := store.GetByQuery(ctx, query)
	assert.NoError(t, errAfter)
	expectedObjectsAfter := []*storage.TestSingleKeyStruct{
		testObjects[1],
		testObjects[3],
	}
	assert.ElementsMatch(t, objectsAfter, expectedObjectsAfter)
}

func TestDeleteByQuery(t *testing.T) {
	testDB := pgtest.ForT(t)
	store := newStore(testDB)

	testObjects := sampleTestSingleKeyStructArray("DeleteByQuery")
	query2 := getMatchFieldQuery("Test Name", "Test DeleteByQuery 2")
	query4 := getMatchFieldQuery("Test Key", "TestDeleteByQuery4")
	query := getDisjunctionQuery(query2, query4)

	queriedObjectsFromEmpty, errQueryFromEmpty := store.GetByQuery(ctx, query)
	assert.NoError(t, errQueryFromEmpty)
	assert.Empty(t, queriedObjectsFromEmpty)

	_, deleteFromEmptyErr := store.DeleteByQuery(ctx, query)
	assert.NoError(t, deleteFromEmptyErr)

	assert.NoError(t, store.UpsertMany(ctx, testObjects))
	for _, obj := range testObjects {
		objBefore, fetchedBefore, errBefore := store.Get(ctx, pkGetter(obj))
		assert.Equal(t, obj, objBefore)
		assert.True(t, fetchedBefore)
		assert.NoError(t, errBefore)
	}

	_, deleteFromPopulatedErr := store.DeleteByQuery(ctx, query)
	assert.NoError(t, deleteFromPopulatedErr)

	for idx, obj := range testObjects {
		objAfter, fetchedAfter, errAfter := store.Get(ctx, pkGetter(obj))
		assert.NoError(t, errAfter)
		if idx == 1 || idx == 3 {
			assert.Nil(t, objAfter)
			assert.False(t, fetchedAfter)
		} else {
			assert.Equal(t, obj, objAfter)
			assert.True(t, fetchedAfter)
		}
	}
}

func TestDeleteByQueryReturningIDs(t *testing.T) {
	testDB := pgtest.ForT(t)
	store := newStore(testDB)

	testObjects := sampleTestSingleKeyStructArray("DeleteByQuery")
	query2 := getMatchFieldQuery("Test Name", "Test DeleteByQuery 2")
	query4 := getMatchFieldQuery("Test Key", "TestDeleteByQuery4")
	query := getDisjunctionQuery(query2, query4)

	queriedObjectsFromEmpty, errQueryFromEmpty := store.GetByQuery(ctx, query)
	assert.NoError(t, errQueryFromEmpty)
	assert.Empty(t, queriedObjectsFromEmpty)

	deletedIDsFromEmpty, deleteFromEmptyErr := store.DeleteByQuery(ctx, query)
	assert.NoError(t, deleteFromEmptyErr)
	assert.Empty(t, deletedIDsFromEmpty)

	assert.NoError(t, store.UpsertMany(ctx, testObjects))
	for _, obj := range testObjects {
		objBefore, fetchedBefore, errBefore := store.Get(ctx, pkGetter(obj))
		assert.Equal(t, obj, objBefore)
		assert.True(t, fetchedBefore)
		assert.NoError(t, errBefore)
	}

	deletedIDsFromPopulated, deleteFromPopulatedErr := store.DeleteByQuery(ctx, query)
	assert.NoError(t, deleteFromPopulatedErr)
	expectedIDs := []string{pkGetter(testObjects[1]), pkGetter(testObjects[3])}
	assert.ElementsMatch(t, deletedIDsFromPopulated, expectedIDs)

	for idx, obj := range testObjects {
		objAfter, fetchedAfter, errAfter := store.Get(ctx, pkGetter(obj))
		assert.NoError(t, errAfter)
		if idx == 1 || idx == 3 {
			assert.Nil(t, objAfter)
			assert.False(t, fetchedAfter)
		} else {
			assert.Equal(t, obj, objAfter)
			assert.True(t, fetchedAfter)
		}
	}
}

type storeCreateFunction = func(testDB *pgtest.TestPostgres) Store[storage.ServiceAccount, *storage.ServiceAccount]

func TestNamespaceScopedPostgresStore(t *testing.T) {
	testClusterNamespaceScopedStoreSAC(
		t,
		resources.ServiceAccount,
		newNamespaceScopedNamespacePostgresStore,
		testutils.GenericNamespaceSACUpsertTestCases,
		testutils.GenericNamespaceSACDeleteTestCases,
		testutils.GenericNamespaceSACGetTestCases,
	)
}

func TestNamespaceScopedCachedStore(t *testing.T) {
	testClusterNamespaceScopedStoreSAC(
		t,
		resources.ServiceAccount,
		newNamespaceScopedNamespaceCachedPostgresStore,
		testutils.GenericNamespaceSACUpsertTestCases,
		testutils.GenericNamespaceSACDeleteTestCases,
		testutils.GenericNamespaceSACGetTestCases,
	)
}

func TestClusterScopedPostgresStore(t *testing.T) {
	testClusterNamespaceScopedStoreSAC(
		t,
		resources.Cluster,
		newClusterScopedNamespacePostgresStore,
		testutils.GenericClusterSACUpsertTestCases,
		testutils.GenericClusterSACDeleteTestCases,
		testutils.GenericClusterSACReadTestCases,
	)
}

func TestClusterScopedCachedStore(t *testing.T) {
	testClusterNamespaceScopedStoreSAC(
		t,
		resources.Cluster,
		newClusterScopedNamespaceCachedPostgresStore,
		testutils.GenericClusterSACUpsertTestCases,
		testutils.GenericClusterSACDeleteTestCases,
		testutils.GenericClusterSACReadTestCases,
	)
}

func testClusterNamespaceScopedStoreSAC(
	t *testing.T,
	scopingResource permissions.ResourceMetadata,
	storeCreate storeCreateFunction,
	getUpsertTestCases func(*testing.T, string) map[string]testutils.SACCrudTestCase,
	getDeleteTestCases func(*testing.T) map[string]testutils.SACCrudTestCase,
	getReadTestCases func(*testing.T) map[string]testutils.SACCrudTestCase,
) {
	testSuite := new(clusterNamespaceScopedStoreSACTestSuite)
	testSuite.storeCreateFunc = storeCreate
	testSuite.scopingResource = scopingResource
	testSuite.getUpsertTestCases = getUpsertTestCases
	testSuite.getDeleteTestCases = getDeleteTestCases
	testSuite.getReadTestCases = getReadTestCases
	suite.Run(t, testSuite)
}

type clusterNamespaceScopedStoreSACTestSuite struct {
	suite.Suite

	storeCreateFunc storeCreateFunction

	scopingResource permissions.ResourceMetadata

	testDB *pgtest.TestPostgres
	store  Store[storage.ServiceAccount, *storage.ServiceAccount]

	getUpsertTestCases func(*testing.T, string) map[string]testutils.SACCrudTestCase
	getDeleteTestCases func(*testing.T) map[string]testutils.SACCrudTestCase
	getReadTestCases   func(*testing.T) map[string]testutils.SACCrudTestCase

	testContexts          map[string]context.Context
	testServiceAccountIDs []string
}

func (s *clusterNamespaceScopedStoreSACTestSuite) SetupTest() {
	s.testDB = pgtest.ForT(s.T())
	s.store = s.storeCreateFunc(s.testDB)
	s.testServiceAccountIDs = make([]string, 0)
	s.testContexts = testutils.GetNamespaceScopedTestContexts(context.Background(), s.T(),
		s.scopingResource)
}

func (s *clusterNamespaceScopedStoreSACTestSuite) TearDownTest() {
	for _, id := range s.testServiceAccountIDs {
		s.deleteServiceAccount(id)
	}
	s.testDB.DB.Close()
}

func (s *clusterNamespaceScopedStoreSACTestSuite) deleteServiceAccount(id string) {
	s.Require().NoError(s.store.Delete(s.testContexts[testutils.UnrestrictedReadWriteCtx], id))
}

func (s *clusterNamespaceScopedStoreSACTestSuite) TestUpsert() {
	testCases := s.getUpsertTestCases(s.T(), testutils.VerbUpsert)
	for name, c := range testCases {
		s.Run(name, func() {
			sa := fixtures.GetScopedServiceAccount(uuid.NewV4().String(), testconsts.Cluster2,
				testconsts.NamespaceB)
			s.testServiceAccountIDs = append(s.testServiceAccountIDs, sa.GetId())
			ctx := s.testContexts[c.ScopeKey]
			err := s.store.Upsert(ctx, sa)
			defer s.deleteServiceAccount(sa.GetId())
			if c.ExpectError {
				s.Require().Error(err)
				s.ErrorIs(err, c.ExpectedError)
			} else {
				s.NoError(err)
			}
			obj, found, err := s.store.Get(s.testContexts[testutils.UnrestrictedReadWriteCtx], sa.GetId())
			s.NoError(err)
			if c.ExpectError {
				s.False(found)
				s.Nil(obj)
			} else {
				s.True(found)
				s.Equal(sa, obj)
			}
		})
	}
}

func (s *clusterNamespaceScopedStoreSACTestSuite) TestUpsertMany() {
	testCases := s.getUpsertTestCases(s.T(), testutils.VerbUpsert)
	for name, c := range testCases {
		s.Run(name, func() {
			sa1 := fixtures.GetScopedServiceAccount(uuid.NewV4().String(), testconsts.Cluster2,
				testconsts.NamespaceB)
			sa2 := fixtures.GetScopedServiceAccount(uuid.NewV4().String(), testconsts.Cluster2,
				testconsts.NamespaceB)
			s.testServiceAccountIDs = append(s.testServiceAccountIDs, sa1.GetId())
			s.testServiceAccountIDs = append(s.testServiceAccountIDs, sa2.GetId())
			ctx := s.testContexts[c.ScopeKey]
			serviceAccounts := []*storage.ServiceAccount{sa1, sa2}
			checkIDs := []string{sa1.GetId(), sa2.GetId()}
			err := s.store.UpsertMany(ctx, serviceAccounts)
			defer s.deleteServiceAccount(sa1.GetId())
			defer s.deleteServiceAccount(sa2.GetId())
			if c.ExpectError {
				s.Require().Error(err)
				s.ErrorIs(err, c.ExpectedError)
			} else {
				s.NoError(err)
			}
			objs, missed, err := s.store.GetMany(s.testContexts[testutils.UnrestrictedReadWriteCtx], checkIDs)
			s.NoError(err)
			if c.ExpectError {
				s.Equal([]int{0, 1}, missed)
				s.Len(objs, 0)
			} else {
				s.Len(missed, 0)
				s.ElementsMatch(serviceAccounts, objs)
			}
		})
	}
}

func (s *clusterNamespaceScopedStoreSACTestSuite) TestDelete() {
	testCases := s.getDeleteTestCases(s.T())
	for name, c := range testCases {
		s.Run(name, func() {
			unrestrictedCtx := s.testContexts[testutils.UnrestrictedReadWriteCtx]
			serviceAccount := fixtures.GetScopedServiceAccount(uuid.NewV4().String(), testconsts.Cluster2,
				testconsts.NamespaceB)
			serviceAccountID := serviceAccount.GetId()
			s.testServiceAccountIDs = append(s.testServiceAccountIDs, serviceAccountID)
			s.Require().NoError(s.store.Upsert(unrestrictedCtx, serviceAccount))
			objBefore, foundBefore, err := s.store.Get(unrestrictedCtx, serviceAccountID)
			s.Require().NoError(err)
			s.Require().True(foundBefore)
			s.Require().Equal(serviceAccount, objBefore)

			ctx := s.testContexts[c.ScopeKey]
			deleteErr := s.store.Delete(ctx, serviceAccountID)
			s.NoError(deleteErr)

			objAfter, foundAfter, checkGetErr := s.store.Get(unrestrictedCtx, serviceAccountID)
			s.NoError(checkGetErr)
			if c.ExpectError {
				s.True(foundAfter)
				s.Equal(serviceAccount, objAfter)
			} else {
				s.False(foundAfter)
				s.Nil(objAfter)
			}
		})
	}
}

func (s *clusterNamespaceScopedStoreSACTestSuite) TestDeleteMany() {
	testCases := s.getDeleteTestCases(s.T())
	for name, c := range testCases {
		s.Run(name, func() {
			unrestrictedCtx := s.testContexts[testutils.UnrestrictedReadWriteCtx]
			serviceAccount1 := fixtures.GetScopedServiceAccount(uuid.NewV4().String(), testconsts.Cluster2,
				testconsts.NamespaceB)
			serviceAccount2 := fixtures.GetScopedServiceAccount(uuid.NewV4().String(), testconsts.Cluster2,
				testconsts.NamespaceB)
			s.testServiceAccountIDs = append(s.testServiceAccountIDs, serviceAccount1.GetId())
			s.testServiceAccountIDs = append(s.testServiceAccountIDs, serviceAccount2.GetId())
			serviceAccounts := []*storage.ServiceAccount{serviceAccount1, serviceAccount2}
			checkIDs := []string{serviceAccount1.GetId(), serviceAccount2.GetId()}
			s.Require().NoError(s.store.UpsertMany(unrestrictedCtx, serviceAccounts))
			objectsBefore, missingBefore, err := s.store.GetMany(unrestrictedCtx, checkIDs)
			s.Require().NoError(err)
			s.Require().Empty(missingBefore)
			s.Require().Equal(serviceAccounts, objectsBefore)

			ctx := s.testContexts[c.ScopeKey]
			deleteErr := s.store.DeleteMany(ctx, checkIDs)
			s.NoError(deleteErr)

			objectsAfter, missingAfter, checkGetErr := s.store.GetMany(unrestrictedCtx, checkIDs)
			s.NoError(checkGetErr)
			if c.ExpectError {
				s.Empty(missingAfter)
				s.Equal(serviceAccounts, objectsAfter)
			} else {
				s.Equal([]int{0, 1}, missingAfter)
				s.Empty(objectsAfter)
			}
		})
	}
}

func (s *clusterNamespaceScopedStoreSACTestSuite) setupReadTest() []*storage.ServiceAccount {
	unrestrictedCtx := s.testContexts[testutils.UnrestrictedReadWriteCtx]
	serviceAccount1 := fixtures.GetScopedServiceAccount(uuid.NewV4().String(), testconsts.Cluster2,
		testconsts.NamespaceB)
	serviceAccount2 := fixtures.GetScopedServiceAccount(uuid.NewV4().String(), testconsts.Cluster2,
		testconsts.NamespaceB)
	serviceAccounts := []*storage.ServiceAccount{serviceAccount1, serviceAccount2}
	s.testServiceAccountIDs = append(s.testServiceAccountIDs, serviceAccount1.GetId())
	s.testServiceAccountIDs = append(s.testServiceAccountIDs, serviceAccount2.GetId())
	s.Require().NoError(s.store.UpsertMany(unrestrictedCtx, serviceAccounts))
	return serviceAccounts
}

func getServiceAccountIDs(serviceAccounts []*storage.ServiceAccount) []string {
	serviceAccountIDs := make([]string, 0, len(serviceAccounts))
	for _, serviceAccount := range serviceAccounts {
		serviceAccountIDs = append(serviceAccountIDs, serviceAccount.GetId())
	}
	return serviceAccountIDs
}

func (s *clusterNamespaceScopedStoreSACTestSuite) TestExists() {
	testCases := s.getReadTestCases(s.T())

	serviceAccounts := s.setupReadTest()
	s.Require().NotEmpty(serviceAccounts)
	serviceAccountID := serviceAccounts[0].GetId()

	for name, c := range testCases {
		s.Run(name, func() {
			ctx := s.testContexts[c.ScopeKey]
			exists, err := s.store.Exists(ctx, serviceAccountID)
			s.NoError(err)
			s.Equal(c.ExpectedFound, exists)
		})
	}
}

func (s *clusterNamespaceScopedStoreSACTestSuite) TestCount() {
	testCases := s.getReadTestCases(s.T())

	serviceAccounts := s.setupReadTest()
	s.Require().NotEmpty(serviceAccounts)

	for name, c := range testCases {
		s.Run(name, func() {
			ctx := s.testContexts[c.ScopeKey]
			count, err := s.store.Count(ctx)
			s.NoError(err)
			if c.ExpectedFound {
				s.Equal(2, count)
			} else {
				s.Equal(0, count)
			}
		})
	}
}

func (s *clusterNamespaceScopedStoreSACTestSuite) TestWalk() {
	testCases := s.getReadTestCases(s.T())

	serviceAccounts := s.setupReadTest()
	s.Require().Len(serviceAccounts, 2)
	expectedFullSerializedServiceAccounts := []string{
		fmt.Sprintf("%s|%s/%s",
			serviceAccounts[0].GetId(),
			serviceAccounts[0].GetClusterId(),
			serviceAccounts[0].GetNamespace(),
		),
		fmt.Sprintf("%s|%s/%s",
			serviceAccounts[1].GetId(),
			serviceAccounts[1].GetClusterId(),
			serviceAccounts[1].GetNamespace(),
		),
	}

	for name, c := range testCases {
		s.Run(name, func() {
			ctx := s.testContexts[c.ScopeKey]
			serializedSAs := make([]string, 0, 2)
			err := s.store.Walk(ctx, func(obj *storage.ServiceAccount) error {
				serializedSAs = append(
					serializedSAs,
					fmt.Sprintf("%s|%s/%s", obj.GetId(), obj.GetClusterId(), obj.GetNamespace()),
				)
				return nil
			})
			s.NoError(err)
			if c.ExpectedFound {
				s.ElementsMatch(serializedSAs, expectedFullSerializedServiceAccounts)
			} else {
				s.Empty(serializedSAs)
			}
		})
	}
}

func (s *clusterNamespaceScopedStoreSACTestSuite) TestWalkByQuery() {
	testCases := s.getReadTestCases(s.T())

	serviceAccounts := s.setupReadTest()
	s.Require().Len(serviceAccounts, 2)
	expectedFullSerializedServiceAccounts := []string{
		fmt.Sprintf("%s|%s/%s",
			serviceAccounts[0].GetId(),
			serviceAccounts[0].GetClusterId(),
			serviceAccounts[0].GetNamespace(),
		),
		fmt.Sprintf("%s|%s/%s",
			serviceAccounts[1].GetId(),
			serviceAccounts[1].GetClusterId(),
			serviceAccounts[1].GetNamespace(),
		),
	}

	for name, c := range testCases {
		s.Run(name, func() {
			ctx := s.testContexts[c.ScopeKey]
			serializedSAs := make([]string, 0, 2)
			err := s.store.WalkByQuery(ctx, search.EmptyQuery(), func(obj *storage.ServiceAccount) error {
				serializedSAs = append(
					serializedSAs,
					fmt.Sprintf("%s|%s/%s", obj.GetId(), obj.GetClusterId(), obj.GetNamespace()),
				)
				return nil
			})
			s.NoError(err)
			if c.ExpectedFound {
				s.ElementsMatch(serializedSAs, expectedFullSerializedServiceAccounts)
			} else {
				s.Empty(serializedSAs)
			}
		})
	}
}

func (s *clusterNamespaceScopedStoreSACTestSuite) TestGetAll() {
	testCases := s.getReadTestCases(s.T())

	serviceAccounts := s.setupReadTest()
	s.Require().NotEmpty(serviceAccounts)

	for name, c := range testCases {
		s.Run(name, func() {
			ctx := s.testContexts[c.ScopeKey]
			fetchedServiceAccounts, err := s.store.GetAll(ctx)
			s.NoError(err)
			if c.ExpectedFound {
				s.ElementsMatch(fetchedServiceAccounts, serviceAccounts)
			} else {
				s.Empty(fetchedServiceAccounts)
			}
		})
	}
}

func (s *clusterNamespaceScopedStoreSACTestSuite) TestGetIDs() {
	testCases := s.getReadTestCases(s.T())

	serviceAccounts := s.setupReadTest()
	s.Require().NotEmpty(serviceAccounts)
	serviceAccountIDs := getServiceAccountIDs(serviceAccounts)

	for name, c := range testCases {
		s.Run(name, func() {
			ctx := s.testContexts[c.ScopeKey]
			fetchedServiceAccountIDs, err := s.store.GetIDs(ctx)
			s.NoError(err)
			if c.ExpectedFound {
				s.ElementsMatch(fetchedServiceAccountIDs, serviceAccountIDs)
			} else {
				s.Empty(fetchedServiceAccountIDs)
			}
		})
	}
}

func (s *clusterNamespaceScopedStoreSACTestSuite) TestGet() {
	testCases := s.getReadTestCases(s.T())

	serviceAccounts := s.setupReadTest()
	s.Require().NotEmpty(serviceAccounts)

	for name, c := range testCases {
		s.Run(name, func() {
			ctx := s.testContexts[c.ScopeKey]
			serviceAccount := serviceAccounts[0]
			obj, found, err := s.store.Get(ctx, serviceAccount.GetId())
			s.NoError(err)
			if c.ExpectedFound {
				s.True(found)
				s.Equal(serviceAccount, obj)
			} else {
				s.False(found)
				s.Nil(obj)
			}
		})
	}
}

func (s *clusterNamespaceScopedStoreSACTestSuite) TestGetMany() {
	testCases := s.getReadTestCases(s.T())

	serviceAccounts := s.setupReadTest()
	s.Require().Len(serviceAccounts, 2)
	serviceAccountIDs := getServiceAccountIDs(serviceAccounts)

	for name, c := range testCases {
		s.Run(name, func() {
			ctx := s.testContexts[c.ScopeKey]
			objects, missing, err := s.store.GetMany(ctx, serviceAccountIDs)
			s.NoError(err)
			if c.ExpectedFound {
				s.Empty(missing)
				s.ElementsMatch(objects, serviceAccounts)
			} else {
				s.Equal([]int{0, 1}, missing)
				s.Empty(objects)
			}
		})
	}
}

func (s *clusterNamespaceScopedStoreSACTestSuite) TestGetByQuery() {
	testCases := s.getReadTestCases(s.T())

	serviceAccounts := s.setupReadTest()
	s.NotEmpty(serviceAccounts)

	for name, c := range testCases {
		s.Run(name, func() {
			ctx := s.testContexts[c.ScopeKey]

			objs, err := s.store.GetByQuery(ctx, search.EmptyQuery())
			s.NoError(err)
			if c.ExpectedFound {
				s.ElementsMatch(objs, serviceAccounts)
			} else {
				s.Nil(objs)
			}
		})
	}
}

func (s *clusterNamespaceScopedStoreSACTestSuite) TestDeleteByQuery() {
	testCases := s.getDeleteTestCases(s.T())

	for name, c := range testCases {
		s.Run(name, func() {
			serviceAccounts := s.setupReadTest()
			s.Len(serviceAccounts, 2)
			serviceAccountIDs := getServiceAccountIDs(serviceAccounts)

			unrestrictedCtx := s.testContexts[testutils.UnrestrictedReadWriteCtx]
			objsBefore, missingBefore, errBefore := s.store.GetMany(unrestrictedCtx, serviceAccountIDs)
			s.NoError(errBefore)
			s.Empty(missingBefore)
			s.ElementsMatch(objsBefore, serviceAccounts)

			ctx := s.testContexts[c.ScopeKey]
			_, err := s.store.DeleteByQuery(ctx, search.EmptyQuery())
			s.NoError(err)

			objsAfter, missingAfter, errAfter := s.store.GetMany(unrestrictedCtx, serviceAccountIDs)
			s.NoError(errAfter)
			if c.ExpectError {
				s.Empty(missingAfter)
				s.ElementsMatch(objsAfter, serviceAccounts)
			} else {
				s.Equal([]int{0, 1}, missingAfter)
				s.Empty(objsAfter)
			}

			s.NoError(s.store.DeleteMany(unrestrictedCtx, serviceAccountIDs))
		})
	}
}

func (s *clusterNamespaceScopedStoreSACTestSuite) TestDeleteByQueryReturningIDs() {
	testCases := s.getDeleteTestCases(s.T())

	for name, c := range testCases {
		s.Run(name, func() {
			serviceAccounts := s.setupReadTest()
			s.Len(serviceAccounts, 2)
			serviceAccountIDs := getServiceAccountIDs(serviceAccounts)

			unrestrictedCtx := s.testContexts[testutils.UnrestrictedReadWriteCtx]
			objsBefore, missingBefore, errBefore := s.store.GetMany(unrestrictedCtx, serviceAccountIDs)
			s.NoError(errBefore)
			s.Empty(missingBefore)
			s.ElementsMatch(objsBefore, serviceAccounts)

			ctx := s.testContexts[c.ScopeKey]
			fetchedIDs, err := s.store.DeleteByQuery(ctx, search.EmptyQuery())
			s.NoError(err)

			objsAfter, missingAfter, errAfter := s.store.GetMany(unrestrictedCtx, serviceAccountIDs)
			s.NoError(errAfter)
			if c.ExpectError {
				s.Empty(fetchedIDs)
				s.Empty(missingAfter)
				s.ElementsMatch(objsAfter, serviceAccounts)
			} else {
				s.ElementsMatch(fetchedIDs, serviceAccountIDs)
				s.Equal([]int{0, 1}, missingAfter)
				s.Empty(objsAfter)
			}

			s.NoError(s.store.DeleteMany(unrestrictedCtx, serviceAccountIDs))
		})
	}
}

// region Helper Functions

var (
	// ClusterScopedServiceAccountsSchema is the go schema for table `service_accounts`.
	ClusterScopedServiceAccountsSchema = func() *walker.Schema {
		schema := walker.Walk(reflect.TypeOf((*storage.ServiceAccount)(nil)), "service_accounts")
		schema.SetOptionsMap(search.Walk(v1.SearchCategory_SERVICE_ACCOUNTS, "serviceaccount", (*storage.ServiceAccount)(nil)))
		schema.ScopingResource = resources.Cluster
		return schema
	}()
)

func newNamespaceScopedNamespacePostgresStore(testDB *pgtest.TestPostgres) Store[storage.ServiceAccount, *storage.ServiceAccount] {
	return NewGenericStore[storage.ServiceAccount, *storage.ServiceAccount](
		testDB.DB,
		pkgSchema.ServiceAccountsSchema,
		serviceAccountPkGetter,
		insertIntoServiceAccounts,
		copyFromServiceAccounts,
		doNothingDurationTimeSetter,
		doNothingDurationTimeSetter,
		isNamespaceScopedUpsertAllowed,
		resources.ServiceAccount,
	)
}

func newNamespaceScopedNamespaceCachedPostgresStore(testDB *pgtest.TestPostgres) Store[storage.ServiceAccount, *storage.ServiceAccount] {
	return NewGenericStoreWithCache[storage.ServiceAccount, *storage.ServiceAccount](
		testDB.DB,
		pkgSchema.ServiceAccountsSchema,
		serviceAccountPkGetter,
		insertIntoServiceAccounts,
		copyFromServiceAccounts,
		doNothingDurationTimeSetter,
		doNothingDurationTimeSetter,
		doNothingDurationTimeSetter,
		isNamespaceScopedUpsertAllowed,
		resources.ServiceAccount,
	)
}

func newClusterScopedNamespacePostgresStore(testDB *pgtest.TestPostgres) Store[storage.ServiceAccount, *storage.ServiceAccount] {
	return NewGenericStore[storage.ServiceAccount, *storage.ServiceAccount](
		testDB.DB,
		ClusterScopedServiceAccountsSchema,
		serviceAccountPkGetter,
		insertIntoServiceAccounts,
		copyFromServiceAccounts,
		doNothingDurationTimeSetter,
		doNothingDurationTimeSetter,
		isClusterScopedUpsertAllowed,
		resources.Cluster,
	)
}

func newClusterScopedNamespaceCachedPostgresStore(testDB *pgtest.TestPostgres) Store[storage.ServiceAccount, *storage.ServiceAccount] {
	return NewGenericStoreWithCache[storage.ServiceAccount, *storage.ServiceAccount](
		testDB.DB,
		ClusterScopedServiceAccountsSchema,
		serviceAccountPkGetter,
		insertIntoServiceAccounts,
		copyFromServiceAccounts,
		doNothingDurationTimeSetter,
		doNothingDurationTimeSetter,
		doNothingDurationTimeSetter,
		isClusterScopedUpsertAllowed,
		resources.Cluster,
	)
}

func serviceAccountPkGetter(obj *storage.ServiceAccount) string {
	return obj.GetId()
}

func isNamespaceScopedUpsertAllowed(ctx context.Context, objs ...*storage.ServiceAccount) error {
	scopeChecker := sac.GlobalAccessScopeChecker(ctx).AccessMode(storage.Access_READ_WRITE_ACCESS).Resource(resources.ServiceAccount)
	if scopeChecker.IsAllowed() {
		return nil
	}
	var deniedIDs []string
	for _, obj := range objs {
		subScopeChecker := scopeChecker.ClusterID(obj.GetClusterId()).Namespace(obj.GetNamespace())
		if !subScopeChecker.IsAllowed() {
			deniedIDs = append(deniedIDs, obj.GetId())
		}
	}
	if len(deniedIDs) != 0 {
		return errors.Wrapf(sac.ErrResourceAccessDenied, "modifying serviceAccounts with IDs [%s] was denied", strings.Join(deniedIDs, ", "))
	}
	return nil
}

func isClusterScopedUpsertAllowed(ctx context.Context, objs ...*storage.ServiceAccount) error {
	scopeChecker := sac.GlobalAccessScopeChecker(ctx).AccessMode(storage.Access_READ_WRITE_ACCESS).Resource(resources.Cluster)
	if scopeChecker.IsAllowed() {
		return nil
	}
	var deniedIDs []string
	for _, obj := range objs {
		subScopeChecker := scopeChecker.ClusterID(obj.GetClusterId())
		if !subScopeChecker.IsAllowed() {
			deniedIDs = append(deniedIDs, obj.GetId())
		}
	}
	if len(deniedIDs) != 0 {
		return errors.Wrapf(sac.ErrResourceAccessDenied, "modifying namespaceMetadatas with IDs [%s] was denied", strings.Join(deniedIDs, ", "))
	}
	return nil
}

func isGlobalScopedUpsertAllowed(ctx context.Context, _ ...*storage.ServiceAccount) error {
	scopeChecker := sac.GlobalAccessScopeChecker(ctx).AccessMode(storage.Access_READ_WRITE_ACCESS).Resource(resources.Access)
	if !scopeChecker.IsAllowed() {
		return sac.ErrResourceAccessDenied
	}
	return nil
}

func insertIntoServiceAccounts(batch *pgx.Batch, obj *storage.ServiceAccount) error {

	serialized, marshalErr := obj.Marshal()
	if marshalErr != nil {
		return marshalErr
	}

	values := []interface{}{
		// parent primary keys start
		pgutils.NilOrUUID(obj.GetId()),
		obj.GetName(),
		obj.GetNamespace(),
		obj.GetClusterName(),
		pgutils.NilOrUUID(obj.GetClusterId()),
		pgutils.EmptyOrMap(obj.GetLabels()),
		pgutils.EmptyOrMap(obj.GetAnnotations()),
		serialized,
	}

	finalStr := "INSERT INTO service_accounts (Id, Name, Namespace, ClusterName, ClusterId, Labels, Annotations, serialized) VALUES($1, $2, $3, $4, $5, $6, $7, $8) ON CONFLICT(Id) DO UPDATE SET Id = EXCLUDED.Id, Name = EXCLUDED.Name, Namespace = EXCLUDED.Namespace, ClusterName = EXCLUDED.ClusterName, ClusterId = EXCLUDED.ClusterId, Labels = EXCLUDED.Labels, Annotations = EXCLUDED.Annotations, serialized = EXCLUDED.serialized"
	batch.Queue(finalStr, values...)

	return nil
}

func copyFromServiceAccounts(ctx context.Context, s Deleter, tx *postgres.Tx, objs ...*storage.ServiceAccount) error {
	batchSize := MaxBatchSize
	if len(objs) < batchSize {
		batchSize = len(objs)
	}
	inputRows := make([][]interface{}, 0, batchSize)

	// This is a copy so first we must delete the rows and re-add them
	// Which is essentially the desired behaviour of an upsert.
	deletes := make([]string, 0, batchSize)

	copyCols := []string{
		"id",
		"name",
		"namespace",
		"clustername",
		"clusterid",
		"labels",
		"annotations",
		"serialized",
	}

	for idx, obj := range objs {
		// Todo: ROX-9499 Figure out how to more cleanly template around this issue.
		log.Debugf("This is here for now because there is an issue with pods_TerminatedInstances where the obj "+
			"in the loop is not used as it only consists of the parent ID and the index.  Putting this here as a stop gap "+
			"to simply use the object.  %s", obj)

		serialized, marshalErr := obj.Marshal()
		if marshalErr != nil {
			return marshalErr
		}

		inputRows = append(inputRows, []interface{}{
			pgutils.NilOrUUID(obj.GetId()),
			obj.GetName(),
			obj.GetNamespace(),
			obj.GetClusterName(),
			pgutils.NilOrUUID(obj.GetClusterId()),
			pgutils.EmptyOrMap(obj.GetLabels()),
			pgutils.EmptyOrMap(obj.GetAnnotations()),
			serialized,
		})

		// Add the ID to be deleted.
		deletes = append(deletes, obj.GetId())

		// if we hit our batch size we need to push the data
		if (idx+1)%batchSize == 0 || idx == len(objs)-1 {
			// copy does not upsert so have to delete first.  parent deletion cascades so only need to
			// delete for the top level parent

			if err := s.DeleteMany(ctx, deletes); err != nil {
				return err
			}
			// clear the inserts and vals for the next batch
			deletes = deletes[:0]

			if _, err := tx.CopyFrom(ctx, pgx.Identifier{"service_accounts"}, copyCols, pgx.CopyFromRows(inputRows)); err != nil {
				return err
			}
			// clear the input rows for the next batch
			inputRows = inputRows[:0]
		}
	}

	return nil
}

func newStore(testDB *pgtest.TestPostgres) Store[storage.TestSingleKeyStruct, *storage.TestSingleKeyStruct] {
	return NewGenericStore[storage.TestSingleKeyStruct, *storage.TestSingleKeyStruct](
		testDB.DB,
		pkgSchema.TestSingleKeyStructsSchema,
		pkGetter,
		insertIntoTestSingleKeyStructs,
		copyFromTestSingleKeyStructs,
		doNothingDurationTimeSetter,
		doNothingDurationTimeSetter,
		GloballyScopedUpsertChecker[storage.TestSingleKeyStruct, *storage.TestSingleKeyStruct](resources.Namespace),
		resources.Namespace,
	)
}

func newTestSingleKeyStruct(key string, name string, intVal int64) *storage.TestSingleKeyStruct {
	return &storage.TestSingleKeyStruct{
		Key:   key,
		Name:  name,
		Int64: intVal,
	}
}

func sampleTestSingleKeyStructArray(pattern string) []*storage.TestSingleKeyStruct {
	output := make([]*storage.TestSingleKeyStruct, 0, 5)
	for i := 1; i <= 5; i++ {
		key := fmt.Sprintf("Test%s%d", pattern, i)
		name := fmt.Sprintf("Test %s %d", pattern, i)
		output = append(output, &storage.TestSingleKeyStruct{Key: key, Name: name, Int64: int64(i)})
	}
	return output
}

func getMatchFieldQuery(fieldName string, value string) *v1.Query {
	return &v1.Query{
		Query: &v1.Query_BaseQuery{
			BaseQuery: &v1.BaseQuery{
				Query: &v1.BaseQuery_MatchFieldQuery{
					MatchFieldQuery: &v1.MatchFieldQuery{
						Field:     fieldName,
						Value:     value,
						Highlight: false,
					},
				},
			},
		},
	}
}

func getDisjunctionQuery(q1 *v1.Query, q2 *v1.Query) *v1.Query {
	if q1 == nil && q2 == nil {
		return nil
	}
	if q1 == nil {
		return q2
	}
	if q2 == nil {
		return q1
	}
	return &v1.Query{
		Query: &v1.Query_Disjunction{
			Disjunction: &v1.DisjunctionQuery{
				Queries: []*v1.Query{q1, q2},
			},
		},
	}
}

// copied from tools/generate-helpers/pg-table-bindings/test/postgres/store.go
func pkGetter(obj *storage.TestSingleKeyStruct) string {
	return obj.GetKey()
}

// copied from tools/generate-helpers/pg-table-bindings/test/postgres/store.go
func insertIntoTestSingleKeyStructs(batch *pgx.Batch, obj *storage.TestSingleKeyStruct) error {

	serialized, marshalErr := obj.Marshal()
	if marshalErr != nil {
		return marshalErr
	}

	values := []interface{}{
		// parent primary keys start
		obj.GetKey(),
		obj.GetName(),
		obj.GetStringSlice(),
		obj.GetBool(),
		obj.GetUint64(),
		obj.GetInt64(),
		obj.GetFloat(),
		pgutils.EmptyOrMap(obj.GetLabels()),
		pgutils.NilOrTime(obj.GetTimestamp()),
		obj.GetEnum(),
		obj.GetEnums(),
		serialized,
	}

	finalStr := "INSERT INTO test_single_key_structs (Key, Name, StringSlice, Bool, Uint64, Int64, Float, Labels, Timestamp, Enum, Enums, serialized) VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12) ON CONFLICT(Key) DO UPDATE SET Key = EXCLUDED.Key, Name = EXCLUDED.Name, StringSlice = EXCLUDED.StringSlice, Bool = EXCLUDED.Bool, Uint64 = EXCLUDED.Uint64, Int64 = EXCLUDED.Int64, Float = EXCLUDED.Float, Labels = EXCLUDED.Labels, Timestamp = EXCLUDED.Timestamp, Enum = EXCLUDED.Enum, Enums = EXCLUDED.Enums, serialized = EXCLUDED.serialized"
	batch.Queue(finalStr, values...)

	return nil
}

// copied from tools/generate-helpers/pg-table-bindings/test/postgres/store.go
func copyFromTestSingleKeyStructs(ctx context.Context, s Deleter, tx *postgres.Tx, objs ...*storage.TestSingleKeyStruct) error {
	batchSize := MaxBatchSize
	if len(objs) < batchSize {
		batchSize = len(objs)
	}
	inputRows := make([][]interface{}, 0, batchSize)

	// This is a copy, so first we must delete the rows and re-add them
	// Which is essentially the desired behaviour of an upsert.
	deletes := make([]string, 0, batchSize)

	copyCols := []string{
		"key",
		"name",
		"stringslice",
		"bool",
		"uint64",
		"int64",
		"float",
		"labels",
		"timestamp",
		"enum",
		"enums",
		"serialized",
	}

	for idx, obj := range objs {
		// Todo: ROX-9499 Figure out how to more cleanly template around this issue.
		log.Debugf("This is here for now because there is an issue with pods_TerminatedInstances where the obj "+
			"in the loop is not used as it only consists of the parent ID and the index.  Putting this here as a stop gap "+
			"to simply use the object.  %s", obj)

		serialized, marshalErr := obj.Marshal()
		if marshalErr != nil {
			return marshalErr
		}

		inputRows = append(inputRows, []interface{}{
			obj.GetKey(),
			obj.GetName(),
			obj.GetStringSlice(),
			obj.GetBool(),
			obj.GetUint64(),
			obj.GetInt64(),
			obj.GetFloat(),
			pgutils.EmptyOrMap(obj.GetLabels()),
			pgutils.NilOrTime(obj.GetTimestamp()),
			obj.GetEnum(),
			obj.GetEnums(),
			serialized,
		})

		// Add the ID to be deleted.
		deletes = append(deletes, obj.GetKey())

		// if we hit our batch size we need to push the data
		if (idx+1)%batchSize == 0 || idx == len(objs)-1 {
			// copy does not upsert so have to delete first.  parent deletion cascades so only need to
			// delete for the top level parent

			if err := s.DeleteMany(ctx, deletes); err != nil {
				return err
			}
			// clear the inserts and values for the next batch
			deletes = deletes[:0]

			if _, err := tx.CopyFrom(ctx, pgx.Identifier{"test_single_key_structs"}, copyCols, pgx.CopyFromRows(inputRows)); err != nil {
				return err
			}
			// clear the input rows for the next batch
			inputRows = inputRows[:0]
		}
	}

	return nil
}

// endregion
