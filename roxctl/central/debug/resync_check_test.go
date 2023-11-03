package debug

import (
	"context"
	"net"
	"testing"
	"time"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/roxctl/common/environment/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

var (
	alert1 = &storage.ListAlert{
		Id:    "id1",
		State: storage.ViolationState_ACTIVE,
	}
	alert2 = &storage.ListAlert{
		Id:    "id2",
		State: storage.ViolationState_ACTIVE,
	}
	alert1Resolved = &storage.ListAlert{
		Id:    "id1",
		State: storage.ViolationState_RESOLVED,
	}
)

type mockAlertService struct {
	v1.UnimplementedAlertServiceServer
	firstCall  []*storage.ListAlert
	secondCall []*storage.ListAlert

	timesCalled int
}

func (m *mockAlertService) ListAlerts(_ context.Context, _ *v1.ListAlertsRequest) (*v1.ListAlertsResponse, error) {
	if m.timesCalled == 0 {
		m.timesCalled++
		return &v1.ListAlertsResponse{
			Alerts: m.firstCall,
		}, nil
	}
	return &v1.ListAlertsResponse{
		Alerts: m.secondCall,
	}, nil
}

type mockPolicyService struct {
	v1.UnimplementedPolicyServiceServer
}

func (m *mockPolicyService) ReassessPolicies(_ context.Context, _ *v1.Empty) (*v1.Empty, error) {
	return &v1.Empty{}, nil
}

func createMockAlertService(t *testing.T, firstCallAlerts, secondCallAlerts []*storage.ListAlert) (*grpc.ClientConn, func()) {
	buffer := 1024 * 1024
	listener := bufconn.Listen(buffer)

	server := grpc.NewServer()
	v1.RegisterAlertServiceServer(server,
		&mockAlertService{firstCall: firstCallAlerts, secondCall: secondCallAlerts})
	v1.RegisterPolicyServiceServer(server,
		&mockPolicyService{})

	go func() {
		utils.IgnoreError(func() error { return server.Serve(listener) })
	}()

	conn, err := grpc.DialContext(context.Background(), "", grpc.WithContextDialer(func(ctx context.Context, s string) (net.Conn, error) {
		return listener.Dial()
	}), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)

	closeFunction := func() {
		utils.IgnoreError(listener.Close)
		server.Stop()
	}

	return conn, closeFunction
}

func Test_GetAlertsDelta(t *testing.T) {
	testCases := map[string]struct {
		firstCallAlerts, secondCallAlerts []*storage.ListAlert
		hasWarning                        bool
		warningMessage                    string
	}{
		"Both lists same: no warnings": {
			firstCallAlerts:  []*storage.ListAlert{alert1},
			secondCallAlerts: []*storage.ListAlert{alert1},
			hasWarning:       false,
		},
		"List sizes different: show warning": {
			firstCallAlerts:  []*storage.ListAlert{alert1},
			secondCallAlerts: []*storage.ListAlert{alert1, alert2},
			hasWarning:       true,
			warningMessage:   "Number of alerts differ in before and after",
		},
		"Same list sizes with different content: show warning": {
			firstCallAlerts:  []*storage.ListAlert{alert1},
			secondCallAlerts: []*storage.ListAlert{alert1Resolved},
			hasWarning:       true,
			warningMessage:   "Alerts content differ in before and after",
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			conn, closeFn := createMockAlertService(t, testCase.firstCallAlerts, testCase.secondCallAlerts)
			defer closeFn()

			mockEnv, _, errOut := mocks.NewEnvWithConn(conn, t)
			cmd, err := commandWithConnection(mockEnv, time.Microsecond, time.Minute, "")
			require.NoError(t, err, "should not fail creating resync command with mock environment")

			_, _, err = cmd.run()
			require.NoError(t, err, "should not fail when calling run() on command")

			if testCase.hasWarning {
				assert.Contains(t, errOut.String(), testCase.warningMessage)
			}
		})
	}
}
