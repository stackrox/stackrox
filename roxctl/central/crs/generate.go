package crs

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
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

const (
	minimumCentralVersionWithConfigurableCrsValidity = "4.9"
)

// generateCRS generates a new CRS using Central's API and writes the newly generated CRS into the
// file specified by `outFilename` (if it is non-empty) or to stdout (if `outFilename` is empty).
func generateCRS(cliEnvironment environment.Environment, name string,
	outFilename string, timeout time.Duration, retryTimeout time.Duration,
	validFor time.Duration, validUntil time.Time,
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
	metadataSvc := v1.NewMetadataServiceClient(conn)
	clusterInitSvc := v1.NewClusterInitServiceClient(conn)

	// pre-4.8 Central silently ignores validUntil/validFor settings.
	// Let us make sure that we fail gracefully instead of silently ignoring these user settings.
	if !validUntil.IsZero() || validFor != 0 {
		centralVersion, err := getCentralXYVersion(ctx, metadataSvc)
		if err != nil {
			return errors.Wrap(err, "retrieving Central version")
		}

		versionRequirementSatisfied, err := versionLessOrEqual(minimumCentralVersionWithConfigurableCrsValidity, centralVersion)
		if err != nil {
			return errors.Wrap(err, "comparing Central version")
		}
		if !versionRequirementSatisfied {
			return errors.Errorf("Central version %s does not support configurable validity periods for CRSs", centralVersion)
		}
	}

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
	if validFor != 0 {
		req.ValidFor = durationpb.New(validFor)
	}
	if !validUntil.IsZero() {
		req.ValidUntil = timestamppb.New(validUntil)
	}
	resp, err := clusterInitSvc.GenerateCRS(ctx, &req)
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
	var validFor string
	var validUntil string

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
			validForDuration := time.Duration(0)
			validUntilTime := time.Time{}
			var err error
			if validFor != "" {
				validForDuration, err = time.ParseDuration(validFor)
				if err != nil {
					return errors.Wrap(err, "Invalid validity duration specified using `--valid-for'")
				}
			}
			if validUntil != "" {
				validUntilTime, err = time.Parse(time.RFC3339, validUntil)
				if err != nil {
					return errors.Wrap(err, "Invalid validity timestamp specified using `--valid-until'")
				}
			}
			return generateCRS(cliEnvironment, name, outputFile, flags.Timeout(cmd), flags.RetryTimeout(cmd), validForDuration, validUntilTime)
		},
	}
	c.PersistentFlags().StringVarP(&validFor, "valid-for", "", "", "Specify validity duration for the new CRS (e.g. \"10m\", \"1d\").")
	c.PersistentFlags().StringVarP(&validUntil, "valid-until", "", "", "Specify validity as an RFC3339 timestamp for the new CRS.")
	c.PersistentFlags().StringVarP(&outputFile, "output", "o", "", "File to be used for storing the newly generated CRS (- for stdout).")
	c.MarkFlagsMutuallyExclusive("valid-for", "valid-until")

	return c
}

func getCentralXYVersion(ctx context.Context, metadataSvc v1.MetadataServiceClient) (string, error) {
	resp, err := metadataSvc.GetMetadata(ctx, &v1.Empty{})
	if err != nil {
		return "", errors.Wrap(err, "retrieving Central metadata")
	}
	fullVersion := resp.GetVersion()
	xyVersion := extractXYVersion(fullVersion)
	return xyVersion, nil
}

func extractXYVersion(version string) string {
	parts := strings.SplitN(version, ".", 3)
	if len(parts) < 2 {
		return version
	}
	return parts[0] + "." + parts[1]
}

func versionLessOrEqual(versionA, versionB string) (bool, error) {
	vA, err := semver.NewVersion(versionA)
	if err != nil {
		return false, err
	}
	vB, err := semver.NewVersion(versionB)
	if err != nil {
		return false, err
	}
	return vA.LessThanEqual(vB), nil
}
