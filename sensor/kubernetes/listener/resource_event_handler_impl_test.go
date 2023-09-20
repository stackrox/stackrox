package listener

import (
	"context"
	"fmt"
	"testing"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
	mocks2 "github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component/mocks"
	"github.com/stackrox/rox/sensor/kubernetes/listener/resources/mocks"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
	"k8s.io/apimachinery/pkg/types"
)

type contextKey int

const (
	ctxKeyTest contextKey = iota
)

func TestResourceEventHandlerImpl(t *testing.T) {
	suite.Run(t, new(ResourceEventHandlerImplTestSuite))
}

type ResourceEventHandlerImplTestSuite struct {
	suite.Suite

	dispatcher *mocks.MockDispatcher
	resolver   *mocks2.MockResolver

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
	suite.resolver = mocks2.NewMockResolver(suite.mockCtrl)
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
	suite.dispatcher.EXPECT().ProcessEvent(obj, nil, central.ResourceAction_SYNC_RESOURCE).
		Return(&component.ResourceEvent{})
	suite.resolver.EXPECT().Send(gomock.Any())
	handler.OnAdd(obj, false)
	suite.Equal(*expectedMap, handler.seenIDs)
}

func (suite *ResourceEventHandlerImplTestSuite) assertFinished(handler *resourceEventHandlerImpl) {
	suite.Nil(handler.missingInitialIDs)
	suite.Nil(handler.seenIDs)
	suite.True(handler.hasSeenAllInitialIDsSignal.IsDone())
}

func (suite *ResourceEventHandlerImplTestSuite) newHandlerImpl() *resourceEventHandlerImpl {
	return suite.newHandlerImplWithContext(context.Background())
}

func (suite *ResourceEventHandlerImplTestSuite) newHandlerImplWithContext(ctx context.Context) *resourceEventHandlerImpl {
	var treatCreatesAsUpdates concurrency.Flag
	treatCreatesAsUpdates.Set(true)
	var eventLock sync.Mutex
	return &resourceEventHandlerImpl{
		context:          ctx,
		eventLock:        &eventLock,
		dispatcher:       suite.dispatcher,
		resolver:         suite.resolver,
		syncingResources: &treatCreatesAsUpdates,

		hasSeenAllInitialIDsSignal: concurrency.NewSignal(),
		seenIDs:                    make(map[types.UID]struct{}),
		missingInitialIDs:          nil,
	}
}

// Test that when a message is handled by an unsynced resourceEventHandlerImpl it's ID is added to the seedIDs set
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

func (suite *ResourceEventHandlerImplTestSuite) TestContextIsPassed() {
	ctx := context.WithValue(context.Background(), ctxKeyTest, "abc")
	handler := suite.newHandlerImplWithContext(ctx)

	obj := randomID()
	suite.dispatcher.EXPECT().ProcessEvent(obj, nil, central.ResourceAction_SYNC_RESOURCE).
		Return(&component.ResourceEvent{})
	suite.resolver.EXPECT().Send(matchContextValue("abc"))
	handler.OnAdd(obj, false)
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

	suite.dispatcher.EXPECT().ProcessEvent(testMsgTwo, nil, central.ResourceAction_SYNC_RESOURCE).
		Return(&component.ResourceEvent{})
	suite.resolver.EXPECT().Send(gomock.Any())
	handler.OnAdd(testMsgTwo, false)
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

func matchContextValue(value string) gomock.Matcher {
	return &contextMatcher{value}
}

type contextMatcher struct {
	value string
}

// Matches implements gomock.Matcher
func (m *contextMatcher) Matches(x interface{}) bool {
	event, ok := x.(*component.ResourceEvent)
	if !ok {
		return false
	}
	return event.Context.Value(ctxKeyTest) == m.value
}

// String implements gomock.Matcher
func (m *contextMatcher) String() string {
	return fmt.Sprintf("received context should have value %s", m.value)
}

var _ gomock.Matcher = &contextMatcher{}
