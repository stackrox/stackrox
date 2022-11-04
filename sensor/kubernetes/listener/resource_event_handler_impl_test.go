package listener

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/uuid"
	mocks2 "github.com/stackrox/rox/sensor/kubernetes/eventpipeline/output/mocks"
	"github.com/stackrox/rox/sensor/kubernetes/listener/resources/mocks"
	"github.com/stretchr/testify/suite"
	"k8s.io/apimachinery/pkg/types"
)

func TestResourceEventHandlerImpl(t *testing.T) {
	suite.Run(t, new(ResourceEventHandlerImplTestSuite))
}

type ResourceEventHandlerImplTestSuite struct {
	suite.Suite

	dispatcher *mocks.MockDispatcher

	mockCtrl *gomock.Controller
}

type hasAnID struct {
	id types.UID
}

func (h *hasAnID) GetUID() types.UID {
	return h.id
}

func (suite *ResourceEventHandlerImplTestSuite) SetupTest() {
	suite.mockCtrl = gomock.NewController(suite.T())
	suite.dispatcher = mocks.NewMockDispatcher(suite.mockCtrl)
}

func (suite *ResourceEventHandlerImplTestSuite) TearDownTest() {
	suite.mockCtrl.Finish()
}

func randomID() *hasAnID {
	return &hasAnID{id: types.UID(uuid.NewV4().String())}
}

func makeExpectedMap(expectedIDs ...*hasAnID) *map[types.UID]struct{} {
	expectedMap := make(map[types.UID]struct{})
	for _, obj := range expectedIDs {
		expectedMap[obj.GetUID()] = struct{}{}
	}
	return &expectedMap
}

func (suite *ResourceEventHandlerImplTestSuite) addObj(handler *resourceEventHandlerImpl, obj *hasAnID, expectedMap *map[types.UID]struct{}) {
	suite.dispatcher.EXPECT().ProcessEvent(obj, nil, central.ResourceAction_SYNC_RESOURCE)
	handler.OnAdd(obj)
	suite.Equal(*expectedMap, handler.seenIDs)
}

func (suite *ResourceEventHandlerImplTestSuite) assertFinished(handler *resourceEventHandlerImpl) {
	suite.Nil(handler.missingInitialIDs)
	suite.Nil(handler.seenIDs)
	suite.True(handler.hasSeenAllInitialIDsSignal.IsDone())
}

func (suite *ResourceEventHandlerImplTestSuite) newHandlerImpl() *resourceEventHandlerImpl {
	var treatCreatesAsUpdates concurrency.Flag
	treatCreatesAsUpdates.Set(true)
	var eventLock sync.Mutex
	return &resourceEventHandlerImpl{
		eventLock:        &eventLock,
		dispatcher:       suite.dispatcher,
		outputQueue:      mocks2.NewMockQueue(),
		syncingResources: &treatCreatesAsUpdates,

		hasSeenAllInitialIDsSignal: concurrency.NewSignal(),
		seenIDs:                    make(map[types.UID]struct{}),
		missingInitialIDs:          nil,
	}
}

//// Test that when a message is handled by an unsynced resourceEventHandlerImpl it's ID is added to the seedIDs set
func (suite *ResourceEventHandlerImplTestSuite) TestIDsAddedToSyncSet() {
	handler := suite.newHandlerImpl()

	testMsgOne := randomID()
	expectedMap := makeExpectedMap(testMsgOne)
	suite.addObj(handler, testMsgOne, expectedMap)

	testMsgTwo := randomID()
	(*expectedMap)[testMsgTwo.GetUID()] = struct{}{}
	suite.addObj(handler, testMsgTwo, expectedMap)

	suite.Empty(handler.missingInitialIDs)
}

func (suite *ResourceEventHandlerImplTestSuite) TestIDsAddedToMissingSet() {
	handler := suite.newHandlerImpl()

	testMsgOne := randomID()
	handler.PopulateInitialObjects([]interface{}{testMsgOne})
	expectedMap := makeExpectedMap(testMsgOne)
	suite.Equal(*expectedMap, handler.missingInitialIDs)

	// Can't change the expected values.  We've already stopped tracking the seen values
	testMsgTwo := randomID()
	handler.PopulateInitialObjects([]interface{}{testMsgTwo})
	suite.Equal(*expectedMap, handler.missingInitialIDs)
}

func (suite *ResourceEventHandlerImplTestSuite) TestIDsRemovedFromMissingSet() {
	handler := suite.newHandlerImpl()

	testMsgOne := randomID()
	testMsgTwo := randomID()
	expectedMap := *makeExpectedMap(testMsgOne, testMsgTwo)
	handler.PopulateInitialObjects([]interface{}{testMsgOne, testMsgTwo})
	suite.Equal(expectedMap, handler.missingInitialIDs)

	var nilMap map[types.UID]struct{}
	suite.addObj(handler, testMsgOne, &nilMap)

	delete(expectedMap, testMsgOne.GetUID())
	suite.Equal(expectedMap, handler.missingInitialIDs)
}

func (suite *ResourceEventHandlerImplTestSuite) TestSeenIDsNotAddedToMissingSet() {
	handler := suite.newHandlerImpl()

	testMsgOne := randomID()
	expectedMap := makeExpectedMap(testMsgOne)
	suite.addObj(handler, testMsgOne, expectedMap)

	testMsgTwo := randomID()
	expectedMap = makeExpectedMap(testMsgTwo)
	handler.PopulateInitialObjects([]interface{}{testMsgOne, testMsgTwo})
	suite.Equal(*expectedMap, handler.missingInitialIDs)
}

func (suite *ResourceEventHandlerImplTestSuite) TestSeenIDsNotUpdatedAfterPopulateInitialObjects() {
	handler := suite.newHandlerImpl()

	testMsgOne := randomID()
	expectedMap := makeExpectedMap(testMsgOne)
	handler.PopulateInitialObjects([]interface{}{testMsgOne, testMsgOne})
	suite.Equal(*expectedMap, handler.missingInitialIDs)

	testMsgTwo := randomID()
	var nilMap map[types.UID]struct{}
	suite.addObj(handler, testMsgTwo, &nilMap)
}

func (suite *ResourceEventHandlerImplTestSuite) TestCompleteSync() {
	handler := suite.newHandlerImpl()

	testMsgOne := randomID()
	testMsgTwo := randomID()
	expectedMap := makeExpectedMap(testMsgOne)
	suite.addObj(handler, testMsgOne, expectedMap)

	expectedMap = makeExpectedMap(testMsgTwo)
	handler.PopulateInitialObjects([]interface{}{testMsgOne, testMsgTwo})
	suite.Equal(*expectedMap, handler.missingInitialIDs)

	suite.dispatcher.EXPECT().ProcessEvent(testMsgTwo, nil, central.ResourceAction_SYNC_RESOURCE)
	handler.OnAdd(testMsgTwo)
	suite.assertFinished(handler)
}

func (suite *ResourceEventHandlerImplTestSuite) TestAllAlreadySeen() {
	handler := suite.newHandlerImpl()

	testMsgOne := randomID()
	expectedMap := makeExpectedMap(testMsgOne)
	suite.addObj(handler, testMsgOne, expectedMap)

	handler.PopulateInitialObjects([]interface{}{testMsgOne})
	suite.assertFinished(handler)
}

func (suite *ResourceEventHandlerImplTestSuite) TestEmptySeenAndEmptyPopulate() {
	handlerOne := suite.newHandlerImpl()
	testMsgOne := randomID()
	suite.addObj(handlerOne, testMsgOne, makeExpectedMap(testMsgOne))

	handlerOne.PopulateInitialObjects([]interface{}{})
	suite.assertFinished(handlerOne)

	handlerTwo := suite.newHandlerImpl()
	handlerTwo.PopulateInitialObjects([]interface{}{})
	suite.assertFinished(handlerTwo)
}
