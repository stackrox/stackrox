package restore

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/protoconv"
	pkgCommon "github.com/stackrox/rox/pkg/roxctl/common"
	"github.com/stackrox/rox/pkg/v2backuprestore"
	"github.com/stackrox/rox/roxctl/common"
	"github.com/stackrox/rox/roxctl/common/flags"
	"github.com/stackrox/rox/roxctl/common/util"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func v2RestoreStatusCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "status",
		Short: "Show information about an active DB restore process",
		Long:  "Show information about an active DB restore process",
		RunE: util.RunENoArgs(func(c *cobra.Command) error {
			return showRestoreStatus(flags.Timeout(c))
		}),
	}

	return c
}

func printStatus(st *v1.DBRestoreProcessStatus) {
	fmt.Println("ID:")
	fmt.Println(" ", st.GetMetadata().GetId())
	fmt.Println("State:")
	fmt.Println(" ", st.GetState())
	fmt.Println("Started at:")
	fmt.Println(" ", protoconv.ConvertTimestampToTimeOrDefault(st.GetMetadata().GetStartTime(), time.Time{}))
	fmt.Println("Started by:")
	fmt.Println(" ", st.GetMetadata().GetInitiatingUserName())
	if lfi := st.GetMetadata().GetHeader().GetLocalFile(); lfi != nil {
		fmt.Println("Source file:")
		fmt.Printf("  %s (%d bytes)\n", lfi.GetPath(), lfi.GetBytesSize())
	}
	payloadSize := v2backuprestore.RestoreBodySize(st.GetMetadata().GetHeader().GetManifest())
	numFiles := len(st.GetMetadata().GetHeader().GetManifest().GetFiles())
	fmt.Println("Transfer progress:")
	fmt.Printf("  %d/%d bytes (%.2f%%); %d/%d files processed\n", st.GetBytesRead(), payloadSize, float32(100*st.GetBytesRead())/float32(payloadSize), st.GetFilesProcessed(), numFiles)
	if errMsg := st.GetError(); errMsg != "" {
		fmt.Println("Error status:")
		fmt.Println(" ", errMsg)
	}
}

func showRestoreStatus(timeout time.Duration) error {
	conn, err := common.GetGRPCConnection()
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
		fmt.Println("No restore process is currently in progress.")
		return nil
	}

	fmt.Println("Active database restore process information")
	fmt.Println("===========================================")
	printStatus(processStatus)

	return nil
}
