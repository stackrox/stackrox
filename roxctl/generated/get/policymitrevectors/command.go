package tmp

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/retry"
	pkgCommon "github.com/stackrox/rox/pkg/roxctl/common"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/printer"
)

var (
	supportedObjectPrinters = []printer.CustomPrinterFactory{
		printer.NewJSONPrinterFactory(false, false),
	}
)

// Command defines the command for GetPolicyMitreVectors operation.
func Command(cliEnvironment environment.Environment) *cobra.Command {
	cmd := &GetPolicyMitreVectorsCommand{env: cliEnvironment}

	objectPrinterFactory, err := printer.NewObjectPrinterFactory("json", supportedObjectPrinters...)
	// should not happen when using default values, must be a programming error
	utils.Must(err)

	c := &cobra.Command{
		Use:   "roxctl get policymitrevectors <id>",
		Short: "Get policymitrevectors with specified ID",
		Long:  "Get policymitrevectors with specified ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			if err := cmd.construct(c, args, objectPrinterFactory); err != nil {
				return err
			}

			if err := cmd.validate(); err != nil {
				return err
			}

			return cmd.run()
		},
	}

	return c
}

// GetPolicyMitreVectorsCommand holds the metadata about the command to get policymitrevectors.
type GetPolicyMitreVectorsCommand struct {
	id                 string
	retryDelay         int
	retryCount         int
	timeout            time.Duration
	env                environment.Environment
	printer            printer.ObjectPrinter
	standardizedFormat bool
}

func (cmd *GetPolicyMitreVectorsCommand) construct(_ *cobra.Command, args []string, f *printer.ObjectPrinterFactory) error {
	cmd.id = args[0]
	p, err := f.CreatePrinter()
	if err != nil {
		return errors.Wrap(err, "could not create printer for image scan result")
	}
	cmd.printer = p
	cmd.standardizedFormat = f.IsStandardizedFormat()

	return nil
}

func (cmd *GetPolicyMitreVectorsCommand) validate() error {
	return nil
}

func (cmd *GetPolicyMitreVectorsCommand) run() error {
	err := retry.WithRetry(func() error {
		_, err := cmd.runHelper()
		return err
	},
		retry.Tries(cmd.retryCount+1),
		retry.OnlyRetryableErrors(),
		retry.OnFailedAttempts(func(err error) {
			cmd.env.Logger().ErrfLn("GetPolicyMitreVectors failed: %v. Retrying after %v seconds...", err, cmd.retryDelay)
			time.Sleep(time.Duration(cmd.retryDelay) * time.Second)
		}),
	)
	if err != nil {
		return errors.Wrapf(err, "GetPolicyMitreVectors request failed after %d retries", cmd.retryCount)
	}
	return nil
}

func (cmd *GetPolicyMitreVectorsCommand) runHelper() (interface{}, error) {
	conn, err := cmd.env.GRPCConnection()
	if err != nil {
		return nil, errors.Wrap(err, "could not establish gRPC connection to Central")
	}
	defer utils.IgnoreError(conn.Close)

	svc := v1.NewPolicyServiceClient(conn)

	ctx, cancel := context.WithTimeout(pkgCommon.Context(), cmd.timeout)
	defer cancel()

	in := &v1.GetPolicyMitreVectorsRequest{
		Id: cmd.id,
	}
	response, err := svc.GetPolicyMitreVectors(ctx, in)
	if err != nil {
		return nil, errors.Wrapf(err, "could not complete request")
	}
	return response, nil
}

func (cmd *GetPolicyMitreVectorsCommand) print(response interface{}) error {
	if err := cmd.printer.Print(response, cmd.env.ColorWriter()); err != nil {
		return errors.Wrap(err, "could not print response")
	}
	return nil
}
