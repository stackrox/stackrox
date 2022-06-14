package mocks

import (
	"bytes"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stackrox/stackrox/roxctl/common/environment"
	"github.com/stackrox/stackrox/roxctl/common/io"
	"github.com/stackrox/stackrox/roxctl/common/printer"
	"google.golang.org/grpc"
)

// NewEnvWithConn creates new environment with given connection. It returns environment and out / errOut buffer.
// It's meant to use in tests only.
func NewEnvWithConn(conn *grpc.ClientConn, t *testing.T) (environment.Environment, *bytes.Buffer, *bytes.Buffer) {
	envMock := NewMockEnvironment(gomock.NewController(t))

	testIO, _, out, errOut := io.TestIO()
	env := environment.NewTestCLIEnvironment(t, testIO, printer.DefaultColorPrinter())

	envMock.EXPECT().InputOutput().AnyTimes().Return(env.InputOutput())
	envMock.EXPECT().Logger().AnyTimes().Return(env.Logger())
	envMock.EXPECT().GRPCConnection().AnyTimes().Return(conn, nil)
	envMock.EXPECT().ColorWriter().AnyTimes().Return(env.ColorWriter())

	return envMock, out, errOut
}
