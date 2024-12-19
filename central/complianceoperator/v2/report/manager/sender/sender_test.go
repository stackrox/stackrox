package sender

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/complianceoperator/v2/report"
	"github.com/stackrox/rox/generated/storage"
	notifierProcessorMocks "github.com/stackrox/rox/pkg/notifier/mocks"
	notifierMocks "github.com/stackrox/rox/pkg/notifiers/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

var (
	results = &report.Results{
		TotalPass:  2,
		TotalFail:  1,
		TotalMixed: 3,
		Clusters:   2,
		Profiles:   []string{"profile-1", "profile-2"},
	}
	notifiers = []*storage.NotifierConfiguration{
		{
			NotifierConfig: &storage.NotifierConfiguration_EmailConfig{
				EmailConfig: &storage.EmailNotifierConfiguration{
					NotifierId:   "notifier-id",
					MailingLists: []string{"mail-1@test.com", "mail-2@test.com"},
				},
			},
		},
	}
)

func TestComplianceReportingSender(t *testing.T) {
	suite.Run(t, new(ComplianceReportingSenderSuite))
}

type ComplianceReportingSenderSuite struct {
	suite.Suite
	ctrl           *gomock.Controller
	processor      *notifierProcessorMocks.MockProcessor
	reportNotifier *notifierMocks.MockReportNotifier
	sender         *ReportSender
}

func (s *ComplianceReportingSenderSuite) Test_SendEmail() {
	ctx := context.Background()
	cases := map[string]struct {
		expectedNotifier  func() any
		expectedErrNotify func() any
		expectErr         bool
	}{
		"send success": {
			expectedNotifier: func() any {
				return s.reportNotifier
			},
			expectedErrNotify: func() any {
				return nil
			},
		},
		"fail getting the notifiers": {
			expectedNotifier: func() any {
				return nil
			},
			expectErr: true,
		},
		"fail on notify": {
			expectedNotifier: func() any {
				return s.reportNotifier
			},
			expectedErrNotify: func() any {
				return errors.New("error")
			},
			expectErr: true,
		},
	}
	for tName, tCase := range cases {
		s.Run(tName, func() {
			if tCase.expectedNotifier != nil {
				s.processor.EXPECT().GetNotifier(gomock.Any(), gomock.Any()).Times(1).Return(tCase.expectedNotifier())
			}
			if tCase.expectedErrNotify != nil {
				s.reportNotifier.EXPECT().ReportNotify(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(tCase.expectedErrNotify())
			}
			errC := s.sender.SendEmail(ctx, "report-name", &bytes.Buffer{}, results, notifiers)
			select {
			case <-time.After(5 * time.Second):
				s.T().Error("timeout waiting for SendEmail to finish")
				s.T().FailNow()
			case err := <-errC:
				if tCase.expectErr {
					assert.Error(s.T(), err)
				} else {
					assert.NoError(s.T(), err)
				}
			}
		})
	}
}

func (s *ComplianceReportingSenderSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.processor = notifierProcessorMocks.NewMockProcessor(s.ctrl)
	s.reportNotifier = notifierMocks.NewMockReportNotifier(s.ctrl)
	s.sender = NewReportSender(s.processor, 0)
}
