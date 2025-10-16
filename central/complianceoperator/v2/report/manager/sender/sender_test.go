package sender

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/complianceoperator/v2/report"
	formatterMocks "github.com/stackrox/rox/central/complianceoperator/v2/report/manager/format/mocks"
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
		storage.NotifierConfiguration_builder{
			EmailConfig: storage.EmailNotifierConfiguration_builder{
				NotifierId:   "notifier-id",
				MailingLists: []string{"mail-1@test.com", "mail-2@test.com"},
			}.Build(),
		}.Build(),
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
	emailFormatter *formatterMocks.MockEmailFormatter
	sender         *ReportSender
}

func (s *ComplianceReportingSenderSuite) Test_SendEmail() {
	ctx := context.Background()
	emailFormatSuccess := func() {
		s.emailFormatter.EXPECT().FormatWithDetails(gomock.Any(), gomock.Any(), gomock.Any()).Times(2).Return("email template", nil)
	}
	cases := map[string]struct {
		expectedNotifier            func() any
		expectedErrNotify           func() any
		expectedEmailFormatterCalls func()
		expectErr                   bool
	}{
		"send success": {
			expectedNotifier: func() any {
				return s.reportNotifier
			},
			expectedErrNotify: func() any {
				return nil
			},
			expectedEmailFormatterCalls: emailFormatSuccess,
		},
		"fail getting the notifiers": {
			expectedNotifier: func() any {
				return nil
			},
			expectErr: true,
		},
		"fail subject formatting": {
			expectedNotifier: func() any {
				return s.reportNotifier
			},
			expectedEmailFormatterCalls: func() {
				gomock.InOrder(
					s.emailFormatter.EXPECT().FormatWithDetails(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return("", errors.New("some error")),
					s.emailFormatter.EXPECT().FormatWithDetails(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return("body template", nil),
				)
			},
			expectedErrNotify: func() any {
				return nil
			},
			expectErr: true,
		},
		"fail body formatting": {
			expectedNotifier: func() any {
				return s.reportNotifier
			},
			expectedEmailFormatterCalls: func() {
				gomock.InOrder(
					s.emailFormatter.EXPECT().FormatWithDetails(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return("subject template", nil),
					s.emailFormatter.EXPECT().FormatWithDetails(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return("", errors.New("some error")),
				)
			},
			expectedErrNotify: func() any {
				return nil
			},
			expectErr: true,
		},
		"fail on notify": {
			expectedNotifier: func() any {
				return s.reportNotifier
			},
			expectedEmailFormatterCalls: emailFormatSuccess,
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
			if tCase.expectedEmailFormatterCalls != nil {
				tCase.expectedEmailFormatterCalls()
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
	s.emailFormatter = formatterMocks.NewMockEmailFormatter(s.ctrl)
	s.sender = NewReportSender(s.processor, 0)
	s.sender.emailFormatter = s.emailFormatter
}
