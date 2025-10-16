package service

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	storageMocks "github.com/stackrox/rox/central/notifier/datastore/mocks"
	_ "github.com/stackrox/rox/central/notifiers/all"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	reporterMocks "github.com/stackrox/rox/pkg/integrationhealth/mocks"
	"github.com/stackrox/rox/pkg/notifier/mocks"
	"github.com/stackrox/rox/pkg/notifiers"
	notifiersMocks "github.com/stackrox/rox/pkg/notifiers/mocks"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/secrets"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/proto"
)

func TestNotifierService(t *testing.T) {
	suite.Run(t, new(notifierServiceTestSuite))
}

type notifierServiceTestSuite struct {
	suite.Suite

	ctrl      *gomock.Controller
	datastore *storageMocks.MockDataStore
	processor *mocks.MockProcessor
	reporter  *reporterMocks.MockReporter

	ctx context.Context
}

func (s *notifierServiceTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.datastore = storageMocks.NewMockDataStore(s.ctrl)
	s.processor = mocks.NewMockProcessor(s.ctrl)
	s.reporter = reporterMocks.NewMockReporter(s.ctrl)
	s.ctx = sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowAllAccessScopeChecker())
}

func (s *notifierServiceTestSuite) getSvc() Service {
	return &serviceImpl{
		storage:   s.datastore,
		processor: s.processor,
		reporter:  s.reporter,
	}
}

func createNotifier(notifierType string) *storage.Notifier {
	generic := &storage.Generic{}
	generic.SetEndpoint("server:25")
	generic.SetSkipTLSVerify(true)
	generic.SetUsername("username")
	generic.SetPassword("password")
	generic.SetAuditLoggingEnabled(false)
	notifier := &storage.Notifier{}
	notifier.SetId("id")
	notifier.SetName("name")
	notifier.SetUiEndpoint("endpoint")
	notifier.SetType(notifierType)
	notifier.SetGeneric(proto.ValueOrDefault(generic))
	return notifier
}

func createUpdateNotifierRequest(notifierType string) *v1.UpdateNotifierRequest {
	unr := &v1.UpdateNotifierRequest{}
	unr.SetNotifier(createNotifier(notifierType))
	return unr
}

func (s *notifierServiceTestSuite) TestPutNotifier() {
	s.datastore.EXPECT().UpdateNotifier(gomock.Any(), gomock.Any()).Return(nil)
	s.processor.EXPECT().UpdateNotifier(gomock.Any(), gomock.Any()).Return()
	_, err := s.getSvc().PutNotifier(s.ctx, &storage.Notifier{})
	s.Error(err)

	_, err = s.getSvc().PutNotifier(s.ctx, createNotifier("generic"))
	s.NoError(err)
}

func (s *notifierServiceTestSuite) TestUpdateNotifier() {
	// We attempt 6 updates below, out of which 3 are successful.
	s.datastore.EXPECT().UpdateNotifier(gomock.Any(), gomock.Any()).Times(3).Return(nil)
	s.processor.EXPECT().UpdateNotifier(gomock.Any(), gomock.Any()).Times(3).Return()

	s.datastore.EXPECT().GetNotifier(gomock.Any(),
		createUpdateNotifierRequest("generic").GetNotifier().GetId()).Return(
		createUpdateNotifierRequest("generic").GetNotifier(), true, nil).AnyTimes()

	_, err := s.getSvc().UpdateNotifier(s.ctx, &v1.UpdateNotifierRequest{})
	s.Error(err)
	updateReq := createUpdateNotifierRequest("generic")
	updateReq.GetNotifier().GetGeneric().SetPassword("updatePassword")
	_, err = s.getSvc().UpdateNotifier(s.ctx, updateReq)
	s.EqualError(err, errors.Wrap(errox.InvalidArgs, "non-zero or unmasked credential field 'Notifier.Notifier_Generic.Generic.Password'").Error())

	updateReq.SetUpdatePassword(true)
	_, err = s.getSvc().UpdateNotifier(s.ctx, updateReq)
	s.NoError(err)

	updateDependentReq := createUpdateNotifierRequest("generic")
	updateDependentReq.GetNotifier().GetGeneric().SetEndpoint("updated-server:25")
	updateDependentReq.SetUpdatePassword(true)
	_, err = s.getSvc().UpdateNotifier(s.ctx, updateDependentReq)
	s.NoError(err)

	secrets.ScrubSecretsFromStructWithReplacement(updateDependentReq, secrets.ScrubReplacementStr)
	updateDependentReq.SetUpdatePassword(false)
	_, err = s.getSvc().UpdateNotifier(s.ctx, updateDependentReq)
	s.EqualError(err, errors.Wrap(errox.InvalidArgs, "credentials required to update field 'Notifier.Notifier_Generic.Generic.Endpoint'").Error())

	updateBasic := createUpdateNotifierRequest("generic")
	updateBasic.GetNotifier().GetGeneric().SetAuditLoggingEnabled(true)
	secrets.ScrubSecretsFromStructWithReplacement(updateBasic, secrets.ScrubReplacementStr)
	_, err = s.getSvc().UpdateNotifier(s.ctx, updateBasic)
	s.NoError(err)
}

func (s *notifierServiceTestSuite) TestNotifierTestNoError() {
	reqNotifier := createNotifier("TestNotifierTestNoError")

	notifiers.Add(reqNotifier.GetType(), func(_ *storage.Notifier) (notifiers.Notifier, error) {
		notifier := notifiersMocks.NewMockNotifier(s.ctrl)
		notifier.EXPECT().Test(s.ctx).Return(nil)
		notifier.EXPECT().Close(s.ctx).Return(nil)

		return notifier, nil
	})

	_, err := s.getSvc().TestNotifier(s.ctx, reqNotifier)
	s.NoError(err)
}

func (s *notifierServiceTestSuite) TestNotifierTestDoesNotExposeInternalErrors() {
	errMsg := "test message"
	baseErrMsg := "127.0.0.1"
	reqNotifier := createNotifier("TestNotifierTestDoesNotExposeInternalErrors")

	notifiers.Add(reqNotifier.GetType(), func(_ *storage.Notifier) (notifiers.Notifier, error) {
		notifier := notifiersMocks.NewMockNotifier(s.ctrl)
		notifier.EXPECT().Test(s.ctx).Return(notifiers.NewNotifierError(errMsg, errors.New(baseErrMsg)))
		notifier.EXPECT().Close(s.ctx).Return(nil)

		return notifier, nil
	})

	_, err := s.getSvc().TestNotifier(s.ctx, reqNotifier)
	s.Error(err)
	s.Assert().Contains(err.Error(), errMsg)
	s.Assert().NotContains(err.Error(), baseErrMsg)
}

func (s *notifierServiceTestSuite) TestNotifierTestUpdatedNoError() {
	reqUpdateNotifier := createUpdateNotifierRequest("TestNotifierTestUpdatedNoError")
	reqUpdateNotifier.SetUpdatePassword(true)

	s.datastore.EXPECT().GetNotifier(gomock.Any(), reqUpdateNotifier.GetNotifier().GetId()).
		Return(reqUpdateNotifier.GetNotifier(), true, nil).AnyTimes()

	notifiers.Add(reqUpdateNotifier.GetNotifier().GetType(), func(_ *storage.Notifier) (notifiers.Notifier, error) {
		notifier := notifiersMocks.NewMockNotifier(s.ctrl)
		notifier.EXPECT().Test(s.ctx).Return(nil)
		notifier.EXPECT().Close(s.ctx).Return(nil)

		return notifier, nil
	})

	_, err := s.getSvc().TestUpdatedNotifier(s.ctx, reqUpdateNotifier)
	s.NoError(err)
}

func (s *notifierServiceTestSuite) TestNotifierTestUpdatedDoesNotExposeInternalErrors() {
	errMsg := "test message"
	baseErrMsg := "127.0.0.1"

	reqUpdateNotifier := createUpdateNotifierRequest("TestNotifierTestUpdatedDoesNotExposeInternalErrors")
	reqUpdateNotifier.SetUpdatePassword(true)

	s.datastore.EXPECT().GetNotifier(gomock.Any(), reqUpdateNotifier.GetNotifier().GetId()).
		Return(reqUpdateNotifier.GetNotifier(), true, nil).AnyTimes()

	notifiers.Add(reqUpdateNotifier.GetNotifier().GetType(), func(_ *storage.Notifier) (notifiers.Notifier, error) {
		notifier := notifiersMocks.NewMockNotifier(s.ctrl)
		notifier.EXPECT().Test(s.ctx).Return(notifiers.NewNotifierError(errMsg, errors.New(baseErrMsg)))
		notifier.EXPECT().Close(s.ctx).Return(nil)

		return notifier, nil
	})

	_, err := s.getSvc().TestUpdatedNotifier(s.ctx, reqUpdateNotifier)
	s.Error(err)
	s.Assert().Contains(err.Error(), errMsg)
	s.Assert().NotContains(err.Error(), baseErrMsg)
}
