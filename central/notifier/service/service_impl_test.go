package service

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	storageMocks "github.com/stackrox/rox/central/notifier/datastore/mocks"
	_ "github.com/stackrox/rox/central/notifiers/all"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	reporterMocks "github.com/stackrox/rox/pkg/integrationhealth/mocks"
	"github.com/stackrox/rox/pkg/notifier/mocks"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/secrets"
	"github.com/stretchr/testify/suite"
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
		Type:       "generic",
		Config: &storage.Notifier_Generic{Generic: &storage.Generic{
			Endpoint:            "server:25",
			SkipTLSVerify:       true,
			Username:            "username",
			Password:            "password",
			AuditLoggingEnabled: false,
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
	// We attempt 6 updates below, out of which 3 are successful.
	s.datastore.EXPECT().UpdateNotifier(gomock.Any(), gomock.Any()).Times(3).Return(nil)
	s.processor.EXPECT().UpdateNotifier(gomock.Any(), gomock.Any()).Times(3).Return()

	s.datastore.EXPECT().GetNotifier(gomock.Any(),
		createUpdateNotifierRequest().GetNotifier().GetId()).Return(
		createUpdateNotifierRequest().GetNotifier(), true, nil).AnyTimes()

	_, err := s.getSvc().UpdateNotifier(s.ctx, &v1.UpdateNotifierRequest{})
	s.Error(err)
	updateReq := createUpdateNotifierRequest()
	updateReq.GetNotifier().GetGeneric().Password = "updatePassword"
	_, err = s.getSvc().UpdateNotifier(s.ctx, updateReq)
	s.EqualError(err, errors.Wrap(errox.InvalidArgs, "non-zero or unmasked credential field 'Notifier.Notifier_Generic.Generic.Password'").Error())

	updateReq.UpdatePassword = true
	_, err = s.getSvc().UpdateNotifier(s.ctx, updateReq)
	s.NoError(err)

	updateDependentReq := createUpdateNotifierRequest()
	updateDependentReq.GetNotifier().GetGeneric().Endpoint = "updated-server:25"
	updateDependentReq.UpdatePassword = true
	_, err = s.getSvc().UpdateNotifier(s.ctx, updateDependentReq)
	s.NoError(err)

	secrets.ScrubSecretsFromStructWithReplacement(updateDependentReq, secrets.ScrubReplacementStr)
	updateDependentReq.UpdatePassword = false
	_, err = s.getSvc().UpdateNotifier(s.ctx, updateDependentReq)
	s.EqualError(err, errors.Wrap(errox.InvalidArgs, "credentials required to update field 'Notifier.Notifier_Generic.Generic.Endpoint'").Error())

	updateBasic := createUpdateNotifierRequest()
	updateBasic.GetNotifier().GetGeneric().AuditLoggingEnabled = true
	secrets.ScrubSecretsFromStructWithReplacement(updateBasic, secrets.ScrubReplacementStr)
	_, err = s.getSvc().UpdateNotifier(s.ctx, updateBasic)
	s.NoError(err)
}
