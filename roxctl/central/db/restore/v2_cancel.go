package restore

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/errox"
	pkgCommon "github.com/stackrox/rox/pkg/roxctl/common"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/flags"
	"github.com/stackrox/rox/roxctl/common/util"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type centralRestoreCancelCommand struct {
	env     environment.Environment
	confirm func() error
	timeout time.Duration
}

func v2RestoreCancelCommand(cliEnvironment environment.Environment) *cobra.Command {
	c := &cobra.Command{
		Use:   "cancel",
		Short: "Cancel the ongoing database restore process.",
		RunE: util.RunENoArgs(func(c *cobra.Command) error {
			return makeCentralRestoreCancelCommand(cliEnvironment, c).cancelActiveRestore()
		}),
	}
	flags.AddForce(c)
	return c
}

func makeCentralRestoreCancelCommand(cliEnvironment environment.Environment, cbr *cobra.Command) *centralRestoreCancelCommand {
	return &centralRestoreCancelCommand{
		env:     cliEnvironment,
		timeout: flags.Timeout(cbr),
		confirm: func() error {
			return flags.CheckConfirmation(cbr, cliEnvironment.Logger(), cliEnvironment.InputOutput())
		},
	}
}

func (cmd *centralRestoreCancelCommand) cancelActiveRestore() error {
	conn, err := cmd.env.GRPCConnection()
	if err != nil {
		return errors.Wrap(err, "could not establish gRPC connection to central")
	}

	ctx, cancel := context.WithTimeout(pkgCommon.Context(), cmd.timeout)
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
		return errox.NotFound.New("no restore process is currently in progress")
	}

	cmd.env.Logger().PrintfLn("Active database restore process information")
	cmd.env.Logger().PrintfLn("===========================================")
	printStatus(cmd.env.Logger(), processStatus)
	cmd.env.Logger().PrintfLn("")
	cmd.env.Logger().PrintfLn("The above restore process will be canceled.")

	if err := cmd.confirm(); err != nil {
		return err
	}

	ctx, cancel = context.WithTimeout(pkgCommon.Context(), cmd.timeout)
	defer cancel()

	_, err = dbClient.CancelRestoreProcess(ctx, &v1.ResourceByID{
		Id: processStatus.GetMetadata().GetId(),
	})

	return err
}
