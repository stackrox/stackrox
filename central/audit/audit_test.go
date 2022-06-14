package audit

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	notifierMocks "github.com/stackrox/stackrox/central/notifier/processor/mocks"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/pkg/grpc/authz/interceptor"
	"github.com/stackrox/stackrox/pkg/sac"
	"github.com/stackrox/stackrox/pkg/sync"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
)

type AuditLogTestSuite struct {
	suite.Suite

	mockCtrl     *gomock.Controller
	notifierMock *notifierMocks.MockProcessor
}

func (suite *AuditLogTestSuite) SetupTest() {
	suite.mockCtrl = gomock.NewController(suite.T())
	suite.notifierMock = notifierMocks.NewMockProcessor(suite.mockCtrl)
}

func (suite *AuditLogTestSuite) TearDownTest() {
	suite.mockCtrl.Finish()
}

func TestAuditLog(t *testing.T) {
	suite.Run(t, new(AuditLogTestSuite))
}

func (suite *AuditLogTestSuite) TestCalculateAuditStatus() {
	var wg sync.WaitGroup

	audit := New(suite.notifierMock)
	interceptorFunc := audit.UnaryServerInterceptor()
	serverInfo := &grpc.UnaryServerInfo{
		Server:     nil,
		FullMethod: "testMethod",
	}
	ctxWithNoAuth := context.Background()
	ctxAuthorised := interceptor.ContextWithAuthStatus(ctxWithNoAuth, nil)

	auditMessage := (*v1.Audit_Message)(nil)
	suite.notifierMock.EXPECT().HasEnabledAuditNotifiers().Times(4).Return(true)
	suite.notifierMock.EXPECT().ProcessAuditMessage(gomock.Any(), gomock.Any()).Times(4).Do(func(ctx context.Context, msg *v1.Audit_Message) {
		auditMessage = msg
		wg.Done()
	})

	// auth error --> AUTH_FAILED
	wg.Add(1)
	_, _ = interceptorFunc(ctxWithNoAuth, nil, serverInfo, handler(nil))
	wg.Wait()
	suite.Equal(v1.Audit_AUTH_FAILED, auditMessage.Status)

	// request error --> REQUEST_FAILED
	wg.Add(1)
	err := errors.New("test error")
	_, _ = interceptorFunc(ctxAuthorised, nil, serverInfo, handler(err))
	wg.Wait()
	suite.Equal(v1.Audit_REQUEST_FAILED, auditMessage.Status)
	suite.Equal("test error", auditMessage.StatusReason)

	// rejected by SAC --> AUTH_FAILED
	wg.Add(1)
	_, _ = interceptorFunc(ctxAuthorised, nil, serverInfo, handler(sac.ErrResourceAccessDenied))
	wg.Wait()
	suite.Equal(v1.Audit_AUTH_FAILED, auditMessage.Status)

	// no error --> REQUEST_SUCCEEDED
	wg.Add(1)
	_, _ = interceptorFunc(ctxAuthorised, nil, serverInfo, handler(nil))
	wg.Wait()
	suite.Equal(v1.Audit_REQUEST_SUCCEEDED, auditMessage.Status)
}

func handler(err error) func(ctx context.Context, req interface{}) (interface{}, error) {
	return func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, err
	}
}
