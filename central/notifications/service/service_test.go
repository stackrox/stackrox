package service

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/pkg/errors"
	dsMocks "github.com/stackrox/rox/central/notifications/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

var (
	errFake          = errors.New("fake error")
	fakeID           = "0925514f-3a33-5931-b431-756406e1a008"
	fakeNotification = &storage.Notification{
		Id:          fakeID,
		Level:       storage.NotificationLevel_NOTIFICATION_LEVEL_DANGER,
		Message:     "message",
		Type:        storage.NotificationType_NOTIFICATION_TYPE_GENERIC,
		Hint:        "hint",
		Domain:      "domain",
		Occurrences: 1,
	}
	maxPagination = &v1.Pagination{
		Limit: math.MaxInt32,
	}
)

func TestNotificationService(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(countNotificationsTestSuite))
	suite.Run(t, new(getNotificationTestSuite))
	suite.Run(t, new(listNotificationsTestSuite))
}

type baseTestSuite struct {
	suite.Suite
	mockCtrl *gomock.Controller

	ctx           context.Context
	datastoreMock *dsMocks.MockDataStore
	service       Service
}

func (s *baseTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.ctx = context.Background()
	s.datastoreMock = dsMocks.NewMockDataStore(s.mockCtrl)
	s.service = newService(s.datastoreMock)
}

func (s *baseTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

// Test CountNotifications

type countNotificationsTestSuite struct {
	baseTestSuite

	fakeCountNotificationsRequest  *v1.CountNotificationsRequest
	fakeCountNotificationsResponse *v1.CountNotificationsResponse
}

func (s *countNotificationsTestSuite) SetupTest() {
	s.baseTestSuite.SetupTest()

	s.fakeCountNotificationsRequest = &v1.CountNotificationsRequest{}
	s.fakeCountNotificationsResponse = &v1.CountNotificationsResponse{Count: 1}
}

func (s *countNotificationsTestSuite) TestCountNotifications_Success() {
	s.datastoreMock.EXPECT().CountNotifications(s.ctx, gomock.Any()).Return(1, nil)
	result, err := s.service.CountNotifications(s.ctx, s.fakeCountNotificationsRequest)

	s.Require().NoError(err)
	s.Equal(s.fakeCountNotificationsResponse, result)
}

func (s *countNotificationsTestSuite) TestCountNotifications_Error() {
	s.datastoreMock.EXPECT().CountNotifications(s.ctx, gomock.Any()).Return(0, errFake)
	result, err := s.service.CountNotifications(s.ctx, s.fakeCountNotificationsRequest)

	s.ErrorIs(err, errFake)
	s.Nil(result)
}

// Test GetNotification

type getNotificationTestSuite struct {
	baseTestSuite

	fakeResourceByIDRequest     *v1.ResourceByID
	fakeGetNotificationResponse *v1.GetNotificationResponse
}

func (s *getNotificationTestSuite) SetupTest() {
	s.baseTestSuite.SetupTest()

	s.fakeResourceByIDRequest = &v1.ResourceByID{Id: fakeID}
	s.fakeGetNotificationResponse = &v1.GetNotificationResponse{
		Notification: convertToServiceType(fakeNotification),
	}
}

func (s *getNotificationTestSuite) TestGetNotification_Success() {
	s.datastoreMock.EXPECT().GetNotificationByID(s.ctx, fakeID).Return(fakeNotification, nil)
	result, err := s.service.GetNotification(s.ctx, s.fakeResourceByIDRequest)

	s.Require().NoError(err)
	s.Equal(s.fakeGetNotificationResponse, result)
}

func (s *getNotificationTestSuite) TestGetNotification_Error() {
	s.datastoreMock.EXPECT().GetNotificationByID(s.ctx, fakeID).Return(nil, errFake)
	result, err := s.service.GetNotification(s.ctx, s.fakeResourceByIDRequest)

	s.ErrorIs(err, errFake)
	s.Nil(result)
}

func (s *getNotificationTestSuite) TestGetNotification_NotFound() {
	s.datastoreMock.EXPECT().GetNotificationByID(s.ctx, fakeID).Return(nil, errox.NotFound)
	result, err := s.service.GetNotification(s.ctx, s.fakeResourceByIDRequest)

	s.ErrorIs(err, errox.NotFound)
	s.Nil(result)
}

// Test ListNotifications

type listNotificationsTestSuite struct {
	baseTestSuite

	fakeStorageNotificationsList []*storage.Notification
	fakeServiceNotificationsList []*v1.Notification
	fakeListNotificationRequest  *v1.ListNotificationsRequest
	fakeListNotificationResponse *v1.ListNotificationsResponse
}

func (s *listNotificationsTestSuite) SetupTest() {
	s.baseTestSuite.SetupTest()

	s.fakeStorageNotificationsList = []*storage.Notification{fakeNotification}
	s.fakeServiceNotificationsList = []*v1.Notification{convertToServiceType(fakeNotification)}
	s.fakeListNotificationRequest = &v1.ListNotificationsRequest{}
	s.fakeListNotificationResponse = &v1.ListNotificationsResponse{Notifications: s.fakeServiceNotificationsList}
}

func (s *listNotificationsTestSuite) TestListNotifications_Success() {
	s.datastoreMock.EXPECT().ListNotifications(s.ctx, gomock.Any()).Return(s.fakeStorageNotificationsList, nil)
	result, err := s.service.ListNotifications(s.ctx, s.fakeListNotificationRequest)

	s.Require().NoError(err)
	s.Equal(s.fakeListNotificationResponse, result)
}

func (s *listNotificationsTestSuite) TestListNotifications_Error() {
	s.datastoreMock.EXPECT().ListNotifications(s.ctx, gomock.Any()).Return(nil, errFake)
	result, err := s.service.ListNotifications(s.ctx, s.fakeListNotificationRequest)

	s.ErrorIs(err, errFake)
	s.Nil(result)
}

// Test query builder

func TestNotificationsQueryBuilder(t *testing.T) {
	t.Parallel()
	filter := &v1.NotificationsFilter{
		From:             protoconv.ConvertTimeToTimestamp(time.Unix(1000, 0)),
		Until:            protoconv.ConvertTimeToTimestamp(time.Unix(10000, 0)),
		Domain:           "domain",
		ResourceType:     "resourceType",
		NotificationType: v1.NotificationType_NOTIFICATION_TYPE_GENERIC,
		Level:            v1.NotificationLevel_NOTIFICATION_LEVEL_DANGER,
	}
	queryBuilder := getQueryBuilderFromFilter(filter)

	rawQuery, err := queryBuilder.RawQuery()
	require.NoError(t, err)

	assert.Contains(t, rawQuery, "Created Time:tr/1000000-10000000")
	assert.Contains(t, rawQuery, `Notification Domain:"domain"`)
	assert.Contains(t, rawQuery, `Notification Level:"NOTIFICATION_LEVEL_DANGER"`)
	assert.Contains(t, rawQuery, `Notification Type:"NOTIFICATION_TYPE_GENERIC"`)
	assert.Contains(t, rawQuery, `Resource Type:"resourceType"`)
}
