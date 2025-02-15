package crs

import (
	"context"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	v1 "github.com/stackrox/rox/generated/api/v1"
	pkgCommon "github.com/stackrox/rox/pkg/roxctl/common"
	"github.com/stackrox/rox/pkg/stringutils"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/roxctl/common"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/flags"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// generateCRS generates a new CRS using Central's API and writes the newly generated CRS into the
// file specified by `outFilename` (if it is non-empty) or to stdout (if `outFilename` is empty).
func generateCRS(cliEnvironment environment.Environment, name string,
	outFilename string, timeout time.Duration, retryTimeout time.Duration,
) error {
	var err error
	var outFile *os.File

	ctx, cancel := context.WithTimeout(pkgCommon.Context(), timeout)
	defer cancel()

	conn, err := cliEnvironment.GRPCConnection(common.WithRetryTimeout(retryTimeout))
	if err != nil {
		return err
	}
	defer utils.IgnoreError(conn.Close)
	svc := v1.NewClusterInitServiceClient(conn)

	outWriter := cliEnvironment.InputOutput().Out()
	if outFilename != "" {
		outFile, err = os.OpenFile(outFilename, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			return errors.Wrap(err, "creating output file for writing CRS")
		}
		outWriter = outFile
		defer func() {
			_ = outFile.Close()
			if err != nil {
				utils.Should(os.Remove(outFilename))
			}
		}()
	}

	req := v1.CRSGenRequest{Name: name}
	resp, err := svc.GenerateCRS(ctx, &req)
	if err != nil {
		if errStatus, ok := status.FromError(err); ok && errStatus.Code() == codes.Unimplemented {
			return errors.Wrap(err, "missing CRS support in Central")
		}
		return errors.Wrap(err, "generating new CRS")
	}

	crs := resp.GetCrs()
	meta := resp.GetMeta()

	cliEnvironment.Logger().InfofLn("Successfully generated new CRS")
	cliEnvironment.Logger().InfofLn("")
	cliEnvironment.Logger().InfofLn("  Name:       %s", meta.GetName())
	cliEnvironment.Logger().InfofLn("  Created at: %s", meta.GetCreatedAt().AsTime().Format(time.RFC3339))
	cliEnvironment.Logger().InfofLn("  Expires at: %s", meta.GetExpiresAt().AsTime().Format(time.RFC3339))
	cliEnvironment.Logger().InfofLn("  Created By: %s", getPrettyUser(meta.GetCreatedBy()))
	cliEnvironment.Logger().InfofLn("  ID:         %s", meta.GetId())

	_, err = outWriter.Write(crs)
	if err != nil {
		return errors.Wrapf(err, "writing CRS to %s", stringutils.FirstNonEmpty(outFilename, "<stdout>"))
	}
	if outFilename != "" {
		cliEnvironment.Logger().InfofLn("The newly generated CRS has been written to file %q.", outFilename)
		if err := outFile.Close(); err != nil {
			return errors.Wrapf(err, "closing output file %q", outFilename)
		}
	}

	cliEnvironment.Logger().InfofLn("Then CRS needs to be stored securely, since it contains secrets.")
	cliEnvironment.Logger().InfofLn("It is not possible to retrieve previously generated CRSs.")
	return nil
}

// generateCommand implements the command for generating new CRSs.
func generateCommand(cliEnvironment environment.Environment) *cobra.Command {
	var outputFile string

	c := &cobra.Command{
		Use:   "generate <CRS name>",
		Short: "Generate a new Cluster Registration Secret",
		Long:  "Generate a new Cluster Registration Secret (CRS) for bootstrapping a new Secured Cluster.",
		Args:  common.ExactArgsWithCustomErrMessage(1, "No name for the CRS specified"),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			if outputFile == "" {
				return common.ErrInvalidCommandOption.New("No output file specified with --output (for stdout, specify '-')")
			}
			if outputFile == "-" {
				outputFile = ""
			}
			return generateCRS(cliEnvironment, name, outputFile, flags.Timeout(cmd), flags.RetryTimeout(cmd))
		},
	}
	c.PersistentFlags().StringVarP(&outputFile, "output", "o", "", "File to be used for storing the newly generated CRS (- for stdout).")

	return c
}
