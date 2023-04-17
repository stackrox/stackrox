package restore

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/protoconv"
	pkgCommon "github.com/stackrox/rox/pkg/roxctl/common"
	"github.com/stackrox/rox/pkg/v2backuprestore"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/flags"
	"github.com/stackrox/rox/roxctl/common/logger"
	"github.com/stackrox/rox/roxctl/common/util"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func v2RestoreStatusCmd(cliEnvironment environment.Environment) *cobra.Command {
	c := &cobra.Command{
		Use:   "status",
		Short: "Show information about the ongoing database restore process.",
		Long:  "Show information such as start time, state, and transfer progress about the ongoing database restore process if one exists.",
		RunE: util.RunENoArgs(func(c *cobra.Command) error {
			return showRestoreStatus(cliEnvironment, flags.Timeout(c))
		}),
	}

	return c
}

func printStatus(logger logger.Logger, st *v1.DBRestoreProcessStatus) {
	logger.PrintfLn("ID:")
	logger.PrintfLn(" ", st.GetMetadata().GetId())
	logger.PrintfLn("State:")
	logger.PrintfLn(" ", st.GetState())
	logger.PrintfLn("Started at:")
	logger.PrintfLn(" ", protoconv.ConvertTimestampToTimeOrDefault(st.GetMetadata().GetStartTime(), time.Time{}))
	logger.PrintfLn("Started by:")
	logger.PrintfLn(" ", st.GetMetadata().GetInitiatingUserName())
	if lfi := st.GetMetadata().GetHeader().GetLocalFile(); lfi != nil {
		logger.PrintfLn("Source file:")
		logger.PrintfLn("  %s (%d bytes)", lfi.GetPath(), lfi.GetBytesSize())
	}
	payloadSize := v2backuprestore.RestoreBodySize(st.GetMetadata().GetHeader().GetManifest())
	numFiles := len(st.GetMetadata().GetHeader().GetManifest().GetFiles())
	logger.PrintfLn("Transfer progress:")
	logger.PrintfLn("  %d/%d bytes (%.2f%%); %d/%d files processed", st.GetBytesRead(), payloadSize, float32(100*st.GetBytesRead())/float32(payloadSize), st.GetFilesProcessed(), numFiles)
	if errMsg := st.GetError(); errMsg != "" {
		logger.PrintfLn("Error status:")
		logger.PrintfLn(" ", errMsg)
	}
}

func showRestoreStatus(cliEnvironment environment.Environment, timeout time.Duration) error {
	conn, err := cliEnvironment.GRPCConnection()
	if err != nil {
		return errors.Wrap(err, "could not establish gRPC connection to central")
	}

	ctx, cancel := context.WithTimeout(pkgCommon.Context(), timeout)
	defer cancel()

	dbClient := v1.NewDBServiceClient(conn)
	activeRestoreProcessResp, err := dbClient.GetActiveRestoreProcess(ctx, &v1.Empty{})
	if err != nil {
		if status.Convert(err).Code() == codes.Unimplemented {
			return ErrV2RestoreNotSupported
		}
		return errors.Wrap(err, "could not get information about active restore process")
	}

	processStatus := activeRestoreProcessResp.GetActiveStatus()
	if processStatus == nil {
		cliEnvironment.Logger().PrintfLn("No restore process is currently in progress.")
		return nil
	}

	cliEnvironment.Logger().PrintfLn("Active database restore process information")
	cliEnvironment.Logger().PrintfLn("===========================================")
	printStatus(cliEnvironment.Logger(), processStatus)

	return nil
}
