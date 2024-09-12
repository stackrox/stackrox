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
)

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

	defer func() {
		if err == nil {
			return
		}
		if outFile == nil {
			return
		}
		name := outFile.Name()
		_ = outFile.Close()
		utils.Should(os.Remove(name))
	}()

	if outFilename != "" {
		outFile, err = os.OpenFile(outFilename, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			return errors.Wrap(err, "opening output file for writing CRS")
		}
	}

	req := v1.CRSGenRequest{Name: name}
	resp, err := svc.GenerateCRS(ctx, &req)
	if err != nil {
		return errors.Wrap(err, "generating new CRS")
	}
	crs := resp.GetCrs()
	meta := resp.GetMeta()

	cliEnvironment.Logger().InfofLn(`Successfully generated new CRS.

  Name:       %s
  Created at: %s
  Expires at: %s
  Created By: %s
  ID:         %s

`,
		meta.GetName(),
		meta.GetCreatedAt().AsTime().Format(time.RFC3339),
		meta.GetExpiresAt().AsTime().Format(time.RFC3339),
		getPrettyUser(meta.GetCreatedBy()),
		meta.GetId())

	outWriter := cliEnvironment.InputOutput().Out()
	if outFile != nil {
		outWriter = outFile
	}
	_, err = outWriter.Write(crs)
	if err != nil {
		return errors.Wrapf(err, "writing CRS to %s", stringutils.FirstNonEmpty(outFilename, "<stdout>"))
	}
	if outFile != nil {
		cliEnvironment.Logger().InfofLn("The newly generated CRS has been written to file %q.", outFile.Name())
		if err := outFile.Close(); err != nil {
			return errors.Wrapf(err, "closing output file %q", outFile.Name())
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
		Use:   "generate <crs name>",
		Short: "Generate a new CRS",
		Long:  "Generate a new CRS for bootstrapping a new StackRox secured cluster",
		Args:  common.ExactArgsWithCustomErrMessage(1, "No name for the CRS specified"),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			if outputFile == "" {
				return common.ErrInvalidCommandOption.New("No output files specified with --output (for stdout, specify '-')")
			}
			if outputFile != "" {
				if outputFile == "-" {
					outputFile = ""
				}
			}
			return generateCRS(cliEnvironment, name, outputFile, flags.Timeout(cmd), flags.RetryTimeout(cmd))
		},
	}
	c.PersistentFlags().StringVarP(&outputFile, "output", "o", "", "File to be used for storing the newly generated CRS (- for stdout)")

	return c
}
