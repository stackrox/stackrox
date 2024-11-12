package grpc

import (
	"bufio"
	"context"
	"net"
	"os"
	"runtime/pprof"
	"strconv"
	"strings"
	"testing"

	grpcMiddleware "github.com/grpc-ecosystem/go-grpc-middleware/v2"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/utils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

// CreateTestGRPCStreamingService creates a streaming server, registers the target
// services there, and returns a connection to the streaming server along with
// a function to close the connection.
func CreateTestGRPCStreamingService(
	ctx context.Context,
	_ testing.TB,
	registerServices func(registrar grpc.ServiceRegistrar),
) (*grpc.ClientConn, func(), error) {
	bufferSize := 1024 * 1024
	listener := bufconn.Listen(bufferSize)

	authInterceptor := func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		return handler(srv, &grpcMiddleware.WrappedServerStream{
			ServerStream:   ss,
			WrappedContext: ctx,
		})
	}

	server := grpc.NewServer(grpc.StreamInterceptor(authInterceptor))
	registerServices(server)

	go func() {
		utils.IgnoreError(func() error { return server.Serve(listener) })
	}()

	conn, err := grpc.DialContext(ctx, "",
		grpc.WithContextDialer(
			func(ctx context.Context, _ string) (net.Conn, error) {
				return listener.DialContext(ctx)
			},
		),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, nil, err
	}

	closeFunc := func() {
		utils.IgnoreError(listener.Close)
		server.Stop()
	}
	return conn, closeFunc, nil
}

var (
	// TODO: Remove this after gathering more information.
	// This is just to log extra information in the server_tests.go if they panic.
	// It should be removed after the investigation is finished.
	printSocketInfo = dummyPrintSocketInfo
	procFiles       = []string{"/proc/net/tcp", "/proc/net/tcp6"}
)

type debugLogger interface {
	Log(args ...any)
	Logf(format string, args ...any)
}

type debugLoggerImpl struct {
	log debugLogger
}

func (d *debugLoggerImpl) Log(args ...any) {
	if d == nil || d.log == nil {
		return
	}
	d.log.Log(args)
}

func (d *debugLoggerImpl) Logf(format string, args ...any) {
	if d == nil || d.log == nil {
		return
	}
	d.log.Logf(format, args)
}

func newDebugLogger(t *testing.T) *debugLoggerImpl {
	return &debugLoggerImpl{
		log: t,
	}
}

func dummyPrintSocketInfo(_ *testing.T) {}

func testPrintSocketInfo(t *testing.T, ports ...uint64) error {
	errList := errorhelpers.NewErrorList("print socket info")
	for _, fName := range procFiles {
		if err := testPrintSocketInfoFromProcFile(t, fName, ports...); err != nil {
			errList.AddError(err)
		}
	}
	return errList.ToError()
}

func testPrintSocketInfoFromProcFile(t *testing.T, fName string, ports ...uint64) (err error) {
	shouldPrintPort := func(port uint64, ports ...uint64) bool {
		for _, p := range ports {
			if p == port {
				return true
			}
		}
		return false
	}
	getStateString := func(code uint64) string {
		codeToState := map[uint64]string{
			0x01: "ESTABLISHED",
			0x02: "SYN_SENT",
			0x03: "SYN_RECV",
			0x04: "FIN_WAIT1",
			0x05: "FIN_WAIT2",
			0x06: "TIME_WAIT",
			0x07: "CLOSE",
			0x08: "CLOSE_WAIT",
			0x09: "LAST_ACK",
			0x0a: "LISTEN",
			0x0b: "CLOSING",
		}
		str, found := codeToState[code]
		if !found {
			return "UNKNOWN"
		}
		return str
	}
	f, openErr := os.Open(fName)
	if openErr != nil {
		return openErr
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil {
			err = closeErr
		}
	}()
	scanner := bufio.NewScanner(f)
	// Ignore the header
	scanner.Scan()
	// Parse the file
	for scanner.Scan() {
		columns := strings.Fields(scanner.Text())
		if len(columns) < 12 {
			return errors.Errorf("not enough columns in the line: %q", scanner.Text())
		}
		fields := strings.Split(columns[1], ":")
		if len(fields) < 2 {
			return errors.Errorf("not enouch fields in the address column: %q", columns[1])
		}
		port, parseErr := strconv.ParseUint(fields[1], 16, 16)
		if parseErr != nil {
			return parseErr
		}
		if !shouldPrintPort(port, ports...) {
			continue
		}
		code, parseErr := strconv.ParseUint(columns[3], 16, 8)
		if parseErr != nil {
			return parseErr
		}
		t.Logf("Port %d is in %q state", port, getStateString(code))
	}
	return err
}

func testPrintStackTraceInfo(_ *testing.T) error {
	errList := errorhelpers.NewErrorList("print stacktrace info")
	for _, p := range pprof.Profiles() {
		if err := p.WriteTo(os.Stderr, 2); err != nil {
			errList.AddError(err)
		}
	}
	return errList.ToError()
}
