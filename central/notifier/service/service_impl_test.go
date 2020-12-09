package service

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	storageMocks "github.com/stackrox/rox/central/notifier/datastore/mocks"
	"github.com/stackrox/rox/central/notifier/processor/mocks"
	_ "github.com/stackrox/rox/central/notifiers/all"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	reporterMocks "github.com/stackrox/rox/pkg/integrationhealth/mocks"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/secrets"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestNotifierService(t *testing.T) {
	t.Parallel()
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

func createNotifier() *storage.Notifier {
	return &storage.Notifier{
		Id:         "id",
		Name:       "name",
		UiEndpoint: "endpoint",
		Type:       "email",
		Config: &storage.Notifier_Email{Email: &storage.Email{
			Server:   "server:25",
			Sender:   "test@stackrox.com",
			Username: "username",
			Password: "password",
		}},
	}
}

func createUpdateNotifierRequest() *v1.UpdateNotifierRequest {
	return &v1.UpdateNotifierRequest{
		Notifier: createNotifier(),
	}
}

func (s *notifierServiceTestSuite) TestPutNotifier() {
	s.datastore.EXPECT().UpdateNotifier(gomock.Any(), gomock.Any()).Return(nil)
	s.processor.EXPECT().UpdateNotifier(gomock.Any(), gomock.Any()).Return()
	_, err := s.getSvc().PutNotifier(s.ctx, &storage.Notifier{})
	s.Error(err)

	_, err = s.getSvc().PutNotifier(s.ctx, createNotifier())
	s.NoError(err)
}

func (s *notifierServiceTestSuite) TestUpdateNotifier() {
	s.datastore.EXPECT().UpdateNotifier(gomock.Any(), gomock.Any()).Return(nil).Times(4)
	s.processor.EXPECT().UpdateNotifier(gomock.Any(), gomock.Any()).Return().Times(4)

	s.datastore.EXPECT().GetNotifier(gomock.Any(),
		createUpdateNotifierRequest().GetNotifier().GetId()).Return(
		createUpdateNotifierRequest().GetNotifier(), true, nil).AnyTimes()

	_, err := s.getSvc().UpdateNotifier(s.ctx, &v1.UpdateNotifierRequest{})
	s.Error(err)
	updateReq := createUpdateNotifierRequest()
	updateReq.GetNotifier().GetEmail().Password = "updatePassword"
	_, err = s.getSvc().UpdateNotifier(s.ctx, updateReq)
	s.Equal(err, status.Error(codes.InvalidArgument, "non-zero or unmasked credential field 'Notifier.Notifier_Email.Email.Password'"))

	updateReq.UpdatePassword = true
	_, err = s.getSvc().UpdateNotifier(s.ctx, updateReq)
	s.NoError(err)

	updateDependentReq := createUpdateNotifierRequest()
	updateDependentReq.GetNotifier().GetEmail().Server = "updated-server:25"
	updateDependentReq.UpdatePassword = true
	_, err = s.getSvc().UpdateNotifier(s.ctx, updateDependentReq)
	s.NoError(err)

	secrets.ScrubSecretsFromStructWithReplacement(updateDependentReq, secrets.ScrubReplacementStr)
	updateDependentReq.UpdatePassword = false
	_, err = s.getSvc().UpdateNotifier(s.ctx, updateDependentReq)
	s.Equal(err, status.Error(codes.InvalidArgument, "credentials required to update field 'Notifier.Notifier_Email.Email.Server'"))

	updateBasic := createUpdateNotifierRequest()
	updateBasic.GetNotifier().GetEmail().From = "support@stackrox.com"
	secrets.ScrubSecretsFromStructWithReplacement(updateBasic, secrets.ScrubReplacementStr)
	_, err = s.getSvc().UpdateNotifier(s.ctx, updateBasic)
	s.NoError(err)
}
