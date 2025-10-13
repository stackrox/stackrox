package mocks

import (
	"bytes"
	"testing"

	"github.com/stackrox/rox/roxctl/common/config"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/io"
	"github.com/stackrox/rox/roxctl/common/printer"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc"
)

// NewEnvWithConn creates a new environment with given connection.
// It returns an environment and out / errOut buffer.
func NewEnvWithConn(conn *grpc.ClientConn, t *testing.T) (environment.Environment, *bytes.Buffer, *bytes.Buffer) {
	envMock := NewMockEnvironment(gomock.NewController(t))

	testIO, _, out, errOut := io.TestIO()
	env := environment.NewTestCLIEnvironment(t, testIO, printer.DefaultColorPrinter())

	envMock.EXPECT().InputOutput().AnyTimes().Return(env.InputOutput())
	envMock.EXPECT().Logger().AnyTimes().Return(env.Logger())
	envMock.EXPECT().GRPCConnection(gomock.Any()).AnyTimes().Return(conn, nil)
	envMock.EXPECT().ColorWriter().AnyTimes().Return(env.ColorWriter())

	return envMock, out, errOut
}

// NewEnv creates a new environment with the given connection and config store.
// It returns an environment and out / errOut buffer.
func NewEnv(conn *grpc.ClientConn, store config.Store, t *testing.T) (environment.Environment, *bytes.Buffer, *bytes.Buffer) {
	envMock := NewMockEnvironment(gomock.NewController(t))

	testIO, _, out, errOut := io.TestIO()
	env := environment.NewTestCLIEnvironment(t, testIO, printer.DefaultColorPrinter())

	envMock.EXPECT().InputOutput().AnyTimes().Return(env.InputOutput())
	envMock.EXPECT().Logger().AnyTimes().Return(env.Logger())
	envMock.EXPECT().GRPCConnection(gomock.Any()).AnyTimes().Return(conn, nil)
	envMock.EXPECT().ColorWriter().AnyTimes().Return(env.ColorWriter())
	envMock.EXPECT().ConfigStore().AnyTimes().Return(store, nil)

	return envMock, out, errOut
}
