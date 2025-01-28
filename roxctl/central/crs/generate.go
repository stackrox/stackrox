package crs

import (
	"context"
	"fmt"
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
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// generateCRS generates a new CRS using Central's API and writes the newly generated CRS into the
// file specified by `outFilename` (if it is non-empty) or to stdout (if `outFilename` is empty).
func generateCRS(cliEnvironment environment.Environment, name string,
	outFilename string, timeout time.Duration, retryTimeout time.Duration,
	maxRegistrations uint64,
	validUntil string, validFor string,
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

	var validUntilTimestamp *timestamppb.Timestamp
	var validForDuration *durationpb.Duration

	if validUntil != "" {
		validUntilTimestamp, err = parseValidUntil(validUntil)
		if err != nil {
			return errors.Wrap(err, "parsing valid-until timestamp")
		}
	}

	if validFor != "" {
		validForDuration, err = parseValidFor(validFor)
		if err != nil {
			return errors.Wrap(err, "parsing valid-for duration")
		}
	}

	req := v1.CRSGenRequest{
		Name:             name,
		MaxRegistrations: maxRegistrations,
		ValidUntil:       validUntilTimestamp,
		ValidFor:         validForDuration,
	}
	resp, err := svc.GenerateCRS(ctx, &req)
	if err != nil {
		if errStatus, ok := status.FromError(err); ok && errStatus.Code() == codes.Unimplemented {
			return errors.Wrap(err, "missing CRS support in Central")
		}
		return errors.Wrap(err, "generating new CRS")
	}

	crs := resp.GetCrs()
	meta := resp.GetMeta()

	registrationLimit := "(no limit)"
	if maxRegistrations := meta.GetMaxRegistrations(); maxRegistrations > 0 {
		registrationLimit = fmt.Sprintf("%d", maxRegistrations)
	}

	cliEnvironment.Logger().InfofLn("Successfully generated new CRS")
	cliEnvironment.Logger().InfofLn("")
	cliEnvironment.Logger().InfofLn("  Name:                          %s", meta.GetName())
	cliEnvironment.Logger().InfofLn("  Created at:                    %s", meta.GetCreatedAt().AsTime().Format(time.RFC3339))
	cliEnvironment.Logger().InfofLn("  Expires at:                    %s", meta.GetExpiresAt().AsTime().Format(time.RFC3339))
	cliEnvironment.Logger().InfofLn("  Created By:                    %s", getPrettyUser(meta.GetCreatedBy()))
	cliEnvironment.Logger().InfofLn("  ID:                            %s", meta.GetId())
	cliEnvironment.Logger().InfofLn("  Allowed cluster registrations: %s", registrationLimit)

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

func parseValidUntil(str string) (*timestamppb.Timestamp, error) {
	if str == "" {
		return nil, errors.New("empty timestamp string")
	}
	timestamp, err := time.Parse(time.RFC3339, str)
	if err != nil {
		return nil, errors.Wrap(err, "parsing timestamp")
	}
	return timestamppb.New(timestamp), nil
}

func parseValidFor(str string) (*durationpb.Duration, error) {
	if str == "" {
		return nil, errors.New("empty duration string")
	}
	duration, err := time.ParseDuration(str)
	if err != nil {
		return nil, errors.Wrap(err, "parsing duration")
	}
	return durationpb.New(duration), nil
}

// generateCommand implements the command for generating new CRSs.
func generateCommand(cliEnvironment environment.Environment) *cobra.Command {
	var outputFile string
	var maxRegistrations uint64
	var validUntil string
	var validFor string

	c := &cobra.Command{
		Use:   "generate --output=<file name> [ --valid-until=<RFC3339 timestamp> | --valid-for=<duration> ] <CRS name>",
		Short: "Generate a new Cluster Registration Secret",
		Long:  "Generate a new Cluster Registration Secret (CRS) for bootstrapping a new Secured Cluster",
		Args:  common.ExactArgsWithCustomErrMessage(1, "No name for the CRS specified"),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			if outputFile == "" {
				return common.ErrInvalidCommandOption.New("No output file specified with --output (for stdout, specify '-')")
			}
			if outputFile == "-" {
				outputFile = ""
			}
			if validUntil != "" && validFor != "" {
				return common.ErrInvalidCommandOption.New("Provide either --valid-until or --valid-for, but not both at the same time")
			}

			return generateCRS(cliEnvironment, name, outputFile, flags.Timeout(cmd), flags.RetryTimeout(cmd), maxRegistrations, validUntil, validFor)
		},
	}
	c.PersistentFlags().StringVarP(&outputFile, "output", "o", "", "File to be used for storing the newly generated CRS (- for stdout)")
	c.PersistentFlags().Uint64Var(&maxRegistrations, "max-registrations", 1, "Specify after how many cluster registrations the CRS will be revoked automatically, use 0 for no limit (default: 1)")
	c.PersistentFlags().StringVarP(&validUntil, "valid-until", "", "", "Specify custom expiration date as an RFC3339 timestamp")
	c.PersistentFlags().StringVarP(&validFor, "valid-for", "", "", "Specify custom validity duration (e.g. '1h30m')")

	return c
}
