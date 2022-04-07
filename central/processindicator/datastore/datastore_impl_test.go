package datastore

import (
	"context"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stackrox/rox/central/analystnotes"
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/central/processindicator"
	"github.com/stackrox/rox/central/processindicator/index"
	indexMocks "github.com/stackrox/rox/central/processindicator/index/mocks"
	"github.com/stackrox/rox/central/processindicator/internal/commentsstore"
	commentsStoreMocks "github.com/stackrox/rox/central/processindicator/internal/commentsstore/mocks"
	"github.com/stackrox/rox/central/processindicator/pruner"
	prunerMocks "github.com/stackrox/rox/central/processindicator/pruner/mocks"
	processSearch "github.com/stackrox/rox/central/processindicator/search"
	searchMocks "github.com/stackrox/rox/central/processindicator/search/mocks"
	"github.com/stackrox/rox/central/processindicator/store"
	storeMocks "github.com/stackrox/rox/central/processindicator/store/mocks"
	rocksStore "github.com/stackrox/rox/central/processindicator/store/rocksdb"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/auth/permissions/utils"
	"github.com/stackrox/rox/pkg/auth/role"
	"github.com/stackrox/rox/pkg/auth/role/resources"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/authn/mocks"
	"github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stackrox/rox/pkg/testutils/rocksdbtest"
	"github.com/stackrox/rox/pkg/testutils/roletest"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
	bolt "go.etcd.io/bbolt"
)

func TestIndicatorDatastore(t *testing.T) {
	suite.Run(t, new(IndicatorDataStoreTestSuite))
}

type IndicatorDataStoreTestSuite struct {
	suite.Suite
	datastore       DataStore
	storage         store.Store
	commentsStorage commentsstore.Store
	indexer         index.Indexer
	searcher        processSearch.Searcher

	rocksDB *rocksdb.RocksDB
	boltDB  *bolt.DB

	hasNoneCtx  context.Context
	hasReadCtx  context.Context
	hasWriteCtx context.Context

	mockCtrl *gomock.Controller
}

func (suite *IndicatorDataStoreTestSuite) SetupTest() {
	suite.hasNoneCtx = sac.WithGlobalAccessScopeChecker(context.Background(), sac.DenyAllAccessScopeChecker())
	suite.hasReadCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Indicator)))
	suite.hasWriteCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Indicator)))

	var err error
	suite.rocksDB = rocksdbtest.RocksDBForT(suite.T())
	suite.NoError(err)
	suite.storage = rocksStore.New(suite.rocksDB)

	suite.boltDB = testutils.DBForSuite(suite)
	suite.commentsStorage = commentsstore.New(suite.boltDB)

	tmpIndex, err := globalindex.TempInitializeIndices("")
	suite.NoError(err)
	suite.indexer = index.New(tmpIndex)

	suite.searcher = processSearch.New(suite.storage, suite.indexer)

	suite.mockCtrl = gomock.NewController(suite.T())
}

func (suite *IndicatorDataStoreTestSuite) TearDownTest() {
	rocksdbtest.TearDownRocksDB(suite.rocksDB)
	testutils.TearDownDB(suite.boltDB)
	suite.mockCtrl.Finish()
}

func (suite *IndicatorDataStoreTestSuite) setupDataStoreNoPruning() {
	var err error
	suite.datastore, err = New(suite.storage, suite.commentsStorage, suite.indexer, suite.searcher, nil)
	suite.Require().NoError(err)
}

func (suite *IndicatorDataStoreTestSuite) setupDataStoreWithMocks() (*storeMocks.MockStore, *indexMocks.MockIndexer, *searchMocks.MockSearcher) {
	mockStorage := storeMocks.NewMockStore(suite.mockCtrl)
	mockStorage.EXPECT().GetKeysToIndex(context.TODO()).Return(nil, nil)

	mockCommentsStorage := commentsStoreMocks.NewMockStore(suite.mockCtrl)

	mockIndexer := indexMocks.NewMockIndexer(suite.mockCtrl)
	mockIndexer.EXPECT().NeedsInitialIndexing().Return(false, nil)

	mockSearcher := searchMocks.NewMockSearcher(suite.mockCtrl)
	var err error
	suite.datastore, err = New(mockStorage, mockCommentsStorage, mockIndexer, mockSearcher, nil)
	suite.Require().NoError(err)

	return mockStorage, mockIndexer, mockSearcher
}

func (suite *IndicatorDataStoreTestSuite) verifyIndicatorsAre(indicators ...*storage.ProcessIndicator) {
	indexResults, err := suite.indexer.Search(search.EmptyQuery())
	suite.NoError(err)
	suite.Len(indexResults, len(indicators))
	resultIDs := make([]string, 0, len(indexResults))
	for _, r := range indexResults {
		resultIDs = append(resultIDs, r.ID)
	}
	indicatorIDs := make([]string, 0, len(indicators))
	for _, i := range indicators {
		indicatorIDs = append(indicatorIDs, i.GetId())
	}
	suite.ElementsMatch(resultIDs, indicatorIDs)

	var foundIndicators []*storage.ProcessIndicator
	err = suite.storage.Walk(suite.hasReadCtx, func(pi *storage.ProcessIndicator) error {
		foundIndicators = append(foundIndicators, pi)
		return nil
	})
	suite.NoError(err)
	suite.ElementsMatch(foundIndicators, indicators)
}

func getIndicators() (indicators []*storage.ProcessIndicator, repeatIndicator *storage.ProcessIndicator) {
	repeatedSignal := &storage.ProcessSignal{
		Args:         "da_args",
		ContainerId:  "aa",
		Name:         "blah",
		ExecFilePath: "file",
	}

	indicators = []*storage.ProcessIndicator{
		{
			Id:            "id1",
			DeploymentId:  "d1",
			PodUid:        "p1",
			ContainerName: "container1",

			Signal: repeatedSignal,
		},
		{
			Id:            "id2",
			DeploymentId:  "d2",
			PodUid:        "p2",
			ContainerName: "container1",

			Signal: &storage.ProcessSignal{
				Name:         "blah",
				Args:         "args2",
				ExecFilePath: "blahpath",
			},
		},
	}

	repeatIndicator = &storage.ProcessIndicator{
		Id:            "id1",
		DeploymentId:  "d1",
		PodUid:        "p1",
		ContainerName: "container1",
		Signal:        repeatedSignal,
	}
	return
}

func (suite *IndicatorDataStoreTestSuite) mustGetCommentsAndValidateCount(ctx context.Context, key *analystnotes.ProcessNoteKey) []*storage.Comment {
	comments, err := suite.datastore.GetCommentsForProcess(ctx, key)
	suite.Require().NoError(err)
	count, err := suite.datastore.GetCommentsCountForProcess(ctx, key)
	suite.Require().NoError(err)
	suite.Len(comments, count)
	return comments
}

func (suite *IndicatorDataStoreTestSuite) ctxWithUIDAndRole(ctx context.Context, userID string, role permissions.ResolvedRole) context.Context {
	identity := mocks.NewMockIdentity(suite.mockCtrl)
	identity.EXPECT().UID().AnyTimes().Return(userID)
	identity.EXPECT().FullName().AnyTimes().Return(userID)
	identity.EXPECT().FriendlyName().AnyTimes().Return(userID)
	identity.EXPECT().User().AnyTimes().Return(nil)
	identity.EXPECT().Roles().AnyTimes().Return([]permissions.ResolvedRole{role})
	identity.EXPECT().Permissions().AnyTimes().Return(role.GetPermissions())

	return authn.ContextWithIdentity(ctx, identity, suite.T())
}

func (suite *IndicatorDataStoreTestSuite) TestComments() {
	suite.setupDataStoreNoPruning()

	roleNone := roletest.NewResolvedRoleWithDenyAll(role.None, nil)
	roleAdmin := roletest.NewResolvedRoleWithDenyAll(role.Admin, utils.FromResourcesWithAccess(resources.AllResourcesModifyPermissions()...))

	uid1Ctx := suite.ctxWithUIDAndRole(suite.hasWriteCtx, "1", roleNone)
	uid2Ctx := suite.ctxWithUIDAndRole(suite.hasWriteCtx, "2", roleNone)
	uid2ButAdminCtx := suite.ctxWithUIDAndRole(suite.hasWriteCtx, "2", roleAdmin)

	indicators, _ := getIndicators()
	suite.NoError(suite.datastore.AddProcessIndicators(uid1Ctx, indicators...))

	key := analystnotes.ProcessToKey(indicators[0])

	_, err := suite.datastore.AddProcessComment(suite.hasNoneCtx, key, &storage.Comment{CommentMessage: "blah"})
	suite.Error(err)
	_, err = suite.datastore.AddProcessComment(suite.hasReadCtx, key, &storage.Comment{CommentMessage: "blah"})
	suite.Error(err)
	id, err := suite.datastore.AddProcessComment(uid1Ctx, key, &storage.Comment{CommentMessage: "blah"})
	suite.NoError(err)
	suite.NotEmpty(id)

	gotComments := suite.mustGetCommentsAndValidateCount(suite.hasNoneCtx, key)
	suite.Empty(gotComments)

	gotComments = suite.mustGetCommentsAndValidateCount(suite.hasReadCtx, key)
	suite.Len(gotComments, 1)
	suite.Equal(id, gotComments[0].GetCommentId())
	suite.Equal("blah", gotComments[0].GetCommentMessage())
	suite.Equal("1", gotComments[0].GetUser().GetId())

	suite.Error(suite.datastore.UpdateProcessComment(suite.hasNoneCtx, key, &storage.Comment{CommentId: id, CommentMessage: "blah2"}))
	suite.Error(suite.datastore.UpdateProcessComment(suite.hasReadCtx, key, &storage.Comment{CommentId: id, CommentMessage: "blah2"}))
	suite.Error(suite.datastore.UpdateProcessComment(uid2Ctx, key, &storage.Comment{CommentId: id, CommentMessage: "blah2"}))
	// Admin cannot edit other people's comments.
	suite.Error(suite.datastore.UpdateProcessComment(uid2ButAdminCtx, key, &storage.Comment{CommentId: id, CommentMessage: "blah2"}))

	suite.NoError(suite.datastore.UpdateProcessComment(uid1Ctx, key, &storage.Comment{CommentId: id, CommentMessage: "blah2"}))

	gotComments = suite.mustGetCommentsAndValidateCount(suite.hasReadCtx, key)
	suite.Len(gotComments, 1)
	suite.Equal(id, gotComments[0].GetCommentId())
	suite.Equal("blah2", gotComments[0].GetCommentMessage())

	suite.Error(suite.datastore.RemoveProcessComment(suite.hasNoneCtx, key, id))
	suite.Error(suite.datastore.RemoveProcessComment(suite.hasReadCtx, key, id))
	suite.Error(suite.datastore.RemoveProcessComment(uid2Ctx, key, id))

	suite.NoError(suite.datastore.RemoveProcessComment(uid1Ctx, key, id))
	gotComments = suite.mustGetCommentsAndValidateCount(suite.hasReadCtx, key)
	suite.Empty(gotComments)

	comment2ID, err := suite.datastore.AddProcessComment(uid1Ctx, key, &storage.Comment{CommentMessage: "blah3"})
	suite.NoError(err)
	suite.NotEmpty(comment2ID)
	gotComments = suite.mustGetCommentsAndValidateCount(suite.hasReadCtx, key)
	suite.Len(gotComments, 1)
	suite.Equal(comment2ID, gotComments[0].GetCommentId())
	suite.Equal("blah3", gotComments[0].GetCommentMessage())
	suite.Equal("1", gotComments[0].GetUser().GetId())
	suite.Error(suite.datastore.RemoveProcessComment(uid2Ctx, key, comment2ID))

	// Admin can delete other people's comments.
	suite.NoError(suite.datastore.RemoveProcessComment(uid2ButAdminCtx, key, comment2ID))

	gotComments = suite.mustGetCommentsAndValidateCount(suite.hasReadCtx, key)
	suite.Empty(gotComments)
}

func (suite *IndicatorDataStoreTestSuite) TestIndicatorBatchAdd() {
	suite.setupDataStoreNoPruning()

	indicators, repeatIndicator := getIndicators()
	suite.NoError(suite.datastore.AddProcessIndicators(suite.hasWriteCtx, append(indicators, repeatIndicator)...))
	suite.verifyIndicatorsAre(indicators[1], repeatIndicator)
}

func (suite *IndicatorDataStoreTestSuite) TestIndicatorBatchAddWithOldIndicator() {
	suite.setupDataStoreNoPruning()

	indicators, repeatIndicator := getIndicators()
	suite.NoError(suite.datastore.AddProcessIndicators(suite.hasWriteCtx, indicators[0]))
	suite.verifyIndicatorsAre(indicators[0])

	suite.NoError(suite.datastore.AddProcessIndicators(suite.hasWriteCtx, indicators[1], repeatIndicator))
	suite.verifyIndicatorsAre(indicators[1], repeatIndicator)
}

func (suite *IndicatorDataStoreTestSuite) TestIndicatorAddOneByOne() {
	suite.setupDataStoreNoPruning()

	indicators, repeatIndicator := getIndicators()
	suite.NoError(suite.datastore.AddProcessIndicators(suite.hasWriteCtx, indicators[0]))
	suite.verifyIndicatorsAre(indicators[0])

	suite.NoError(suite.datastore.AddProcessIndicators(suite.hasWriteCtx, indicators[1]))
	suite.verifyIndicatorsAre(indicators...)

	suite.NoError(suite.datastore.AddProcessIndicators(suite.hasWriteCtx, repeatIndicator))
	suite.verifyIndicatorsAre(indicators[1], repeatIndicator)
}

func generateIndicatorsWithPods(podIDs []string, containerIDs []string) []*storage.ProcessIndicator {
	var indicators []*storage.ProcessIndicator
	for _, p := range podIDs {
		for _, c := range containerIDs {
			indicators = append(indicators, &storage.ProcessIndicator{
				Id:     fmt.Sprintf("indicator_id_%s_%s", p, c),
				PodUid: p,
				Signal: &storage.ProcessSignal{
					ContainerId:  fmt.Sprintf("%s_%s", p, c),
					ExecFilePath: fmt.Sprintf("EXECFILE_%s_%s", p, c),
				},
			})
		}
	}
	return indicators
}

func (suite *IndicatorDataStoreTestSuite) TestIndicatorRemovalByPodID() {
	suite.setupDataStoreNoPruning()

	indicators := generateIndicatorsWithPods([]string{"p1", "p2"}, []string{"c1", "c2"})
	suite.NoError(suite.datastore.AddProcessIndicators(suite.hasWriteCtx, indicators...))
	suite.verifyIndicatorsAre(indicators...)

	suite.NoError(suite.datastore.RemoveProcessIndicatorsByPod(suite.hasWriteCtx, "p1"))
	suite.verifyIndicatorsAre(generateIndicatorsWithPods([]string{"p2"}, []string{"c1", "c2"})...)
}

func (suite *IndicatorDataStoreTestSuite) TestIndicatorRemovalByPodIDAgain() {
	suite.setupDataStoreNoPruning()

	indicators := generateIndicatorsWithPods([]string{"p1", "p2", "p3"}, []string{"c1", "c2", "c3"})
	suite.NoError(suite.datastore.AddProcessIndicators(suite.hasWriteCtx, indicators...))
	suite.verifyIndicatorsAre(indicators...)

	suite.NoError(suite.datastore.RemoveProcessIndicatorsByPod(suite.hasWriteCtx, "pnonexistent"))
	suite.verifyIndicatorsAre(indicators...)
}

func (suite *IndicatorDataStoreTestSuite) TestPruning() {
	const prunePeriod = 100 * time.Millisecond
	mockPrunerFactory := prunerMocks.NewMockFactory(suite.mockCtrl)
	mockPrunerFactory.EXPECT().Period().Return(prunePeriod)
	indicators, _ := getIndicators()

	prunedSignal := concurrency.NewSignal()
	mockPruner := prunerMocks.NewMockPruner(suite.mockCtrl)
	mockPruner.EXPECT().Finish().AnyTimes().Do(func() {
		prunedSignal.Signal()
	})

	pruneTurnstile := concurrency.NewTurnstile()
	mockPrunerFactory.EXPECT().StartPruning().AnyTimes().DoAndReturn(func() pruner.Pruner {
		pruneTurnstile.Wait()
		return mockPruner
	})

	matcher := func(expectedIndicators ...*storage.ProcessIndicator) gomock.Matcher {
		return testutils.PredMatcher(fmt.Sprintf("id with args matcher for %+v", expectedIndicators), func(passed []processindicator.IDAndArgs) bool {
			ourIDsAndArgs := make([]processindicator.IDAndArgs, 0, len(expectedIndicators))
			for _, indicator := range expectedIndicators {
				ourIDsAndArgs = append(ourIDsAndArgs, processindicator.IDAndArgs{ID: indicator.GetId(), Args: indicator.GetSignal().GetArgs()})
			}
			sort.Slice(ourIDsAndArgs, func(i, j int) bool {
				return ourIDsAndArgs[i].ID < ourIDsAndArgs[j].ID
			})
			sort.Slice(passed, func(i, j int) bool {
				return passed[i].ID < passed[j].ID
			})
			if len(ourIDsAndArgs) != len(passed) {
				return false
			}
			for i, idAndArg := range ourIDsAndArgs {
				if idAndArg.ID != passed[i].ID || idAndArg.Args != passed[i].Args {
					return false
				}
			}
			return true
		})
	}
	var err error
	suite.datastore, err = New(suite.storage, suite.commentsStorage, suite.indexer, suite.searcher, mockPrunerFactory)
	suite.Require().NoError(err)
	suite.NoError(suite.datastore.AddProcessIndicators(suite.hasWriteCtx, indicators...))
	suite.verifyIndicatorsAre(indicators...)

	mockPruner.EXPECT().Prune(matcher(indicators...)).Return(nil)
	pruneTurnstile.AllowOne()
	suite.True(concurrency.WaitWithTimeout(&prunedSignal, 3*prunePeriod))
	suite.verifyIndicatorsAre(indicators...)

	// Allow the next prune to go through. However, the prune function should not be
	// called because we should have a cache hit.
	prunedSignal.Reset()
	pruneTurnstile.AllowOne()
	suite.True(concurrency.WaitWithTimeout(&prunedSignal, 3*prunePeriod))
	suite.verifyIndicatorsAre(indicators...)

	// Now add an extra indicator; this should cause a cache miss and we should hit the pruning.
	extraIndicator := indicators[0].Clone()
	extraIndicator.Id = uuid.NewV4().String()
	extraIndicator.Signal.Args = uuid.NewV4().String()
	suite.NoError(suite.datastore.AddProcessIndicators(suite.hasWriteCtx, extraIndicator))

	// Allow the next prune to go through; this time, prune something.
	expectedIndicators := []*storage.ProcessIndicator{extraIndicator}
	expectedIndicators = append(expectedIndicators, indicators...)
	mockPruner.EXPECT().Prune(matcher(expectedIndicators...)).Return([]string{indicators[0].GetId()})
	prunedSignal.Reset()
	pruneTurnstile.AllowOne()
	suite.True(concurrency.WaitWithTimeout(&prunedSignal, 3*prunePeriod))
	expectedIndicators = []*storage.ProcessIndicator{extraIndicator}
	expectedIndicators = append(expectedIndicators, indicators[1:]...)
	suite.verifyIndicatorsAre(expectedIndicators...)

	// Allow the next prune to go through; we should have a cache hit, so no need to mock a call.
	prunedSignal.Reset()
	pruneTurnstile.AllowOne()
	suite.True(concurrency.WaitWithTimeout(&prunedSignal, 3*prunePeriod))

	suite.Len(suite.datastore.(*datastoreImpl).prunedArgsLengthCache, 1)

	// Delete all the indicators.
	uniquePodIDs := set.NewStringSet()
	for _, indicator := range expectedIndicators {
		uniquePodIDs.Add(indicator.GetPodUid())
	}
	for podID := range uniquePodIDs {
		suite.NoError(suite.datastore.RemoveProcessIndicatorsByPod(suite.hasWriteCtx, podID))
	}

	// Allow one more prune through.
	// All the indicators have been deleted, so no call to Prune is expected.
	prunedSignal.Reset()
	pruneTurnstile.AllowOne()
	suite.True(concurrency.WaitWithTimeout(&prunedSignal, 3*prunePeriod))

	// This is not great because we're testing an implementation detail, but whatever.
	// The goal is to make sure that there's no memory leak here.
	suite.Len(suite.datastore.(*datastoreImpl).prunedArgsLengthCache, 0)

	// Close the prune turnstile to allow all the prunes to go through, else the Stop will be blocked on the turnstile.
	pruneTurnstile.Close()

	suite.datastore.Stop()
	suite.True(suite.datastore.Wait(concurrency.Timeout(3 * prunePeriod)))
}

func (suite *IndicatorDataStoreTestSuite) TestEnforcesGet() {
	mockStore, _, _ := suite.setupDataStoreWithMocks()
	mockStore.EXPECT().Get(gomock.Any(), gomock.Any()).Return(&storage.ProcessIndicator{}, true, nil)

	indicator, exists, err := suite.datastore.GetProcessIndicator(suite.hasNoneCtx, "hkjddjhk")
	suite.NoError(err, "expected no error, should return nil without access")
	suite.False(exists)
	suite.Nil(indicator, "expected return value to be nil")
}

func (suite *IndicatorDataStoreTestSuite) TestAllowsGet() {
	mockStore, _, _ := suite.setupDataStoreWithMocks()
	testIndicator := &storage.ProcessIndicator{}

	mockStore.EXPECT().Get(gomock.Any(), gomock.Any()).Return(testIndicator, true, nil)
	indicator, exists, err := suite.datastore.GetProcessIndicator(suite.hasReadCtx, "An Id")
	suite.NoError(err, "expected no error trying to read with permissions")
	suite.True(exists)
	suite.Equal(testIndicator, indicator)

	mockStore.EXPECT().Get(suite.hasWriteCtx, gomock.Any()).Return(testIndicator, true, nil)
	indicator, exists, err = suite.datastore.GetProcessIndicator(suite.hasWriteCtx, "beef")
	suite.NoError(err, "expected no error trying to read with permissions")
	suite.True(exists)
	suite.Equal(testIndicator, indicator)
}

func (suite *IndicatorDataStoreTestSuite) TestEnforcesAdd() {
	storeMock, indexMock, _ := suite.setupDataStoreWithMocks()
	storeMock.EXPECT().UpsertMany(suite.hasWriteCtx, gomock.Any()).Times(0)
	indexMock.EXPECT().AddProcessIndicators(gomock.Any()).Times(0)

	err := suite.datastore.AddProcessIndicators(suite.hasNoneCtx, &storage.ProcessIndicator{})
	suite.Error(err, "expected an error trying to write without permissions")

	err = suite.datastore.AddProcessIndicators(suite.hasReadCtx, &storage.ProcessIndicator{})
	suite.Error(err, "expected an error trying to write without permissions")
}

func (suite *IndicatorDataStoreTestSuite) TestEnforcesAddMany() {
	storeMock, indexMock, _ := suite.setupDataStoreWithMocks()
	storeMock.EXPECT().UpsertMany(suite.hasWriteCtx, gomock.Any()).Times(0)
	indexMock.EXPECT().AddProcessIndicators(gomock.Any()).Times(0)

	err := suite.datastore.AddProcessIndicators(suite.hasNoneCtx, &storage.ProcessIndicator{})
	suite.Error(err, "expected an error trying to write without permissions")

	err = suite.datastore.AddProcessIndicators(suite.hasReadCtx, &storage.ProcessIndicator{})
	suite.Error(err, "expected an error trying to write without permissions")
}

func (suite *IndicatorDataStoreTestSuite) TestAllowsAddMany() {
	storeMock, indexMock, _ := suite.setupDataStoreWithMocks()
	storeMock.EXPECT().UpsertMany(suite.hasWriteCtx, gomock.Any()).Return(nil)
	indexMock.EXPECT().AddProcessIndicators(gomock.Any()).Return(nil)

	storeMock.EXPECT().AckKeysIndexed(suite.hasWriteCtx, "id").Return(nil)

	err := suite.datastore.AddProcessIndicators(suite.hasWriteCtx, &storage.ProcessIndicator{Id: "id"})
	suite.NoError(err, "expected no error trying to write with permissions")
}

func (suite *IndicatorDataStoreTestSuite) TestEnforcesRemoveByPod() {
	_, indexMock, _ := suite.setupDataStoreWithMocks()
	indexMock.EXPECT().DeleteProcessIndicators(gomock.Any()).Times(0)

	err := suite.datastore.RemoveProcessIndicatorsByPod(suite.hasNoneCtx, "Joseph Rules")
	suite.Error(err, "expected an error trying to write without permissions")

	err = suite.datastore.RemoveProcessIndicatorsByPod(suite.hasReadCtx, "nfsiux")
	suite.Error(err, "expected an error trying to write without permissions")
}

func (suite *IndicatorDataStoreTestSuite) TestAllowsRemoveByPod() {
	storeMock, indexMock, searchMock := suite.setupDataStoreWithMocks()
	searchMock.EXPECT().Search(gomock.Any(), gomock.Any()).Return([]search.Result{{ID: "jkldfjk"}}, nil)
	storeMock.EXPECT().DeleteMany(suite.hasWriteCtx, gomock.Any()).Return(nil)
	indexMock.EXPECT().DeleteProcessIndicators(gomock.Any()).Return(nil)

	storeMock.EXPECT().AckKeysIndexed(suite.hasWriteCtx, "jkldfjk").Return(nil)

	err := suite.datastore.RemoveProcessIndicatorsByPod(suite.hasWriteCtx, "eoiurvbf")
	suite.NoError(err, "expected no error trying to write with permissions")
}

func TestProcessIndicatorReindexSuite(t *testing.T) {
	suite.Run(t, new(ProcessIndicatorReindexSuite))
}

type ProcessIndicatorReindexSuite struct {
	suite.Suite

	storage         *storeMocks.MockStore
	commentsStorage *commentsStoreMocks.MockStore
	indexer         *indexMocks.MockIndexer
	searcher        *searchMocks.MockSearcher

	mockCtrl *gomock.Controller
}

func (suite *ProcessIndicatorReindexSuite) SetupTest() {
	suite.mockCtrl = gomock.NewController(suite.T())
	suite.storage = storeMocks.NewMockStore(suite.mockCtrl)
	suite.commentsStorage = commentsStoreMocks.NewMockStore(suite.mockCtrl)
	suite.indexer = indexMocks.NewMockIndexer(suite.mockCtrl)
	suite.searcher = searchMocks.NewMockSearcher(suite.mockCtrl)
}

func (suite *ProcessIndicatorReindexSuite) TestReconciliationPartialReindex() {
	suite.storage.EXPECT().GetKeysToIndex(gomock.Any()).Return([]string{"A", "B", "C"}, nil)
	suite.indexer.EXPECT().NeedsInitialIndexing().Return(false, nil)

	pi1 := fixtures.GetProcessIndicator()
	pi1.Id = "A"
	pi2 := fixtures.GetProcessIndicator()
	pi2.Id = "B"
	pi3 := fixtures.GetProcessIndicator()
	pi3.Id = "C"

	processes := []*storage.ProcessIndicator{pi1, pi2, pi3}

	suite.storage.EXPECT().GetMany(gomock.Any(), []string{"A", "B", "C"}).Return(processes, nil, nil)
	suite.indexer.EXPECT().AddProcessIndicators(processes).Return(nil)
	suite.storage.EXPECT().AckKeysIndexed(gomock.Any(), []string{"A", "B", "C"}).Return(nil)

	_, err := New(suite.storage, suite.commentsStorage, suite.indexer, suite.searcher, nil)
	suite.NoError(err)

	// Make listAlerts just A,B so C should be deleted
	processes = processes[:1]
	suite.storage.EXPECT().GetKeysToIndex(gomock.Any()).Return([]string{"A", "B", "C"}, nil)
	suite.indexer.EXPECT().NeedsInitialIndexing().Return(false, nil)

	suite.storage.EXPECT().GetMany(gomock.Any(), []string{"A", "B", "C"}).Return(processes, []int{2}, nil)
	suite.indexer.EXPECT().AddProcessIndicators(processes).Return(nil)
	suite.indexer.EXPECT().DeleteProcessIndicators([]string{"C"}).Return(nil)
	suite.storage.EXPECT().AckKeysIndexed(gomock.Any(), []string{"A", "B", "C"}).Return(nil)

	_, err = New(suite.storage, suite.commentsStorage, suite.indexer, suite.searcher, nil)
	suite.NoError(err)
}
