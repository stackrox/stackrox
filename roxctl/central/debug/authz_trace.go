package debug

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/golang/protobuf/jsonpb"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	v1 "github.com/stackrox/rox/generated/api/v1"
	pkgCommon "github.com/stackrox/rox/pkg/roxctl/common"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/flags"
	"github.com/stackrox/rox/roxctl/common/util"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	authzTraceTimeout = 20 * time.Minute
)

// authzTraceCommand allows to download authz trace from Central.
func authzTraceCommand(cliEnvironment environment.Environment) *cobra.Command {
	c := &cobra.Command{
		Use:   "authz-trace",
		Short: "Stream built-in authorizer traces for all incoming requests.",
		Long: `Stream built-in authorizer traces for all incoming requests.
The command blocks for the given number of minutes and collects the authorization trace log for all incoming API requests to the Central service.`,
		RunE: util.RunENoArgs(func(c *cobra.Command) error {
			return writeAuthzTraces(cliEnvironment, flags.Timeout(c), flags.RetryTimeout(c))
		}),
	}
	flags.AddTimeoutWithDefault(c, authzTraceTimeout)
	flags.AddRetryTimeoutWithDefault(c, time.Duration(0))
	return c
}

func writeAuthzTraces(cliEnvironment environment.Environment, timeout time.Duration, retryTimeout time.Duration) error {
	// Write traces directly to stdout without buffering. Sync iff supported,
	// e.g., stdout is redirected to a file and not attached to the console.
	traceOutput := os.Stdout //nolint:forbidigo // TODO(ROX-13473)
	toSync := false
	if traceOutput.Sync() == nil {
		toSync = true
	}

	streamErr := streamAuthzTraces(cliEnvironment, timeout, retryTimeout, traceOutput)

	var syncErr error
	if toSync {
		syncErr = traceOutput.Sync()
		if syncErr != nil {
			syncErr = errors.Wrap(syncErr, "syncing stdout")
		}
	}

	return multierror.Append(streamErr, syncErr).ErrorOrNil()
}

func streamAuthzTraces(cliEnvironment environment.Environment, timeout time.Duration,
	retryTimeout time.Duration, traceOutput io.Writer,
) error {
	// pkgCommon.Context() is canceled on SIGINT, we will use that to stop on Ctrl-C.
	ctx, cancel := context.WithTimeout(pkgCommon.Context(), timeout)
	defer cancel()

	conn, err := cliEnvironment.GRPCConnection(retryTimeout)
	if err != nil {
		return err
	}
	defer utils.IgnoreError(conn.Close)

	// Establish authz trace stream from central.
	client := v1.NewDebugServiceClient(conn)
	stream, err := client.StreamAuthzTraces(ctx, &v1.Empty{})
	if err != nil {
		return err
	}

	// Receive authz traces from central, convert them to JSON, and write.
	// We will get an error from stream.Recv() when one of 3 things happen:
	// 1. Timeout is exceeded
	// 2. User presses Ctrl-C
	// 3. Transport layer error
	//
	// When the context times out or is canceled, the stream might return an EOF
	// or (likely) a corresponding gRPC status error.
	for {
		trace, recvErr := stream.Recv()
		if recvErr != nil {
			if errors.Is(recvErr, io.EOF) || status.Code(recvErr) == codes.Canceled || status.Code(recvErr) == codes.DeadlineExceeded {
				return nil
			}
			return recvErr
		}

		if err := (&jsonpb.Marshaler{}).Marshal(traceOutput, trace); err != nil {
			return errors.Wrap(err, "marshaling a trace to JSON")
		}
		if _, err := traceOutput.Write([]byte{'\n'}); err != nil {
			return err
		}
	}
}
