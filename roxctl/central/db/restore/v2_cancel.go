package restore

import (
	"context"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	v1 "github.com/stackrox/rox/generated/api/v1"
	pkgCommon "github.com/stackrox/rox/pkg/roxctl/common"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/flags"
	"github.com/stackrox/rox/roxctl/common/util"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func v2RestoreCancelCommand(cliEnvironment environment.Environment) *cobra.Command {
	c := &cobra.Command{
		Use: "cancel",
		RunE: util.RunENoArgs(func(c *cobra.Command) error {
			return cancelActiveRestore(cliEnvironment, c)
		}),
	}
	flags.AddForce(c)

	return c
}

func cancelActiveRestore(cliEnvironment environment.Environment, c *cobra.Command) error {
	conn, err := cliEnvironment.GRPCConnection()
	if err != nil {
		return errors.Wrap(err, "could not establish gRPC connection to central")
	}

	ctx, cancel := context.WithTimeout(pkgCommon.Context(), flags.Timeout(c))
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
		return errors.New("No restore process is currently in progress")
	}

	cliEnvironment.Logger().PrintfLn("Active database restore process information")
	cliEnvironment.Logger().PrintfLn("===========================================")
	printStatus(cliEnvironment.Logger(), processStatus)
	cliEnvironment.Logger().PrintfLn("")
	cliEnvironment.Logger().PrintfLn("The above restore process will be canceled.")

	if err := flags.CheckConfirmation(c); err != nil {
		return err
	}

	ctx, cancel = context.WithTimeout(pkgCommon.Context(), flags.Timeout(c))
	defer cancel()

	_, err = dbClient.CancelRestoreProcess(ctx, &v1.ResourceByID{
		Id: processStatus.GetMetadata().GetId(),
	})

	return err
}
