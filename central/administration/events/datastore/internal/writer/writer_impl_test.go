package writer

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	storeMocks "github.com/stackrox/rox/central/administration/events/datastore/internal/store/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/administration/events"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

var errFake = errors.New("fake error")

func TestEventsWriter(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(writerTestSuite))
}

type writerTestSuite struct {
	suite.Suite
	mockCtrl *gomock.Controller

	readCtx  context.Context
	writeCtx context.Context
	store    *storeMocks.MockStore
	writer   Writer
}

func (s *writerTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())

	s.readCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Administration),
		),
	)
	s.writeCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Administration),
		),
	)
	s.store = storeMocks.NewMockStore(s.mockCtrl)
	s.writer = New(s.store)
}

func (s *writerTestSuite) TestWriteEvent_Error() {
	event := &events.AdministrationEvent{
		Level:        storage.AdministrationEventLevel_ADMINISTRATION_EVENT_LEVEL_ERROR,
		Message:      "message",
		Type:         storage.AdministrationEventType_ADMINISTRATION_EVENT_TYPE_GENERIC,
		Hint:         "hint",
		Domain:       "domain",
		ResourceID:   "something",
		ResourceType: "something",
	}
	id := "cbb15d73-880a-50c1-97ab-afb71547e925"

	s.store.EXPECT().Get(s.writeCtx, id).Return(nil, false, errFake)
	err := s.writer.Upsert(s.writeCtx, event)
	s.ErrorIs(err, errFake)
}

func (s *writerTestSuite) TestWriteEvent_NilEvent_Error() {
	err := s.writer.Upsert(s.writeCtx, nil)
	s.ErrorIs(err, errox.InvalidArgs)
}
