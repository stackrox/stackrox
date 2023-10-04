package initbundles

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	v1 "github.com/stackrox/rox/generated/api/v1"
	pkgCommon "github.com/stackrox/rox/pkg/roxctl/common"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/roxctl/common"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/flags"
)

func fetchCAConfig(cliEnvironment environment.Environment, outputFile string,
	timeout time.Duration, retryTimeout time.Duration,
) error {
	ctx, cancel := context.WithTimeout(pkgCommon.Context(), timeout)
	defer cancel()

	conn, err := cliEnvironment.GRPCConnection(retryTimeout)
	if err != nil {
		return err
	}
	defer utils.IgnoreError(conn.Close)
	svc := v1.NewClusterInitServiceClient(conn)

	if outputFile == "" {
		return writeCA(ctx, svc, cliEnvironment.InputOutput().Out())
	}

	bundleOutput, err := os.OpenFile(outputFile, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return errors.Wrap(err, "opening output file for writing CA config")
	}
	defer func() {
		if bundleOutput != nil {
			_ = bundleOutput.Close()
			utils.Should(os.Remove(outputFile))
		}
	}()
	if err := writeCA(ctx, svc, bundleOutput); err != nil {
		return err
	}
	cliEnvironment.Logger().InfofLn("The CA configuration has been written to file %q.", outputFile)
	if err := bundleOutput.Close(); err != nil {
		return errors.Wrap(err, "closing output file for CA config")
	}
	return nil
}

func writeCA(ctx context.Context, svc v1.ClusterInitServiceClient, bundleOutput io.Writer) error {
	resp, err := svc.GetCAConfig(ctx, &v1.Empty{})
	if err != nil {
		return errors.Wrap(err, "fetching CA config")
	}

	_, err = bundleOutput.Write(resp.GetHelmValuesBundle())
	if err != nil {
		return errors.Wrap(err, "writing init bundle")
	}
	return nil
}

func fetchCACommand(cliEnvironment environment.Environment) *cobra.Command {
	var outputFile string

	c := &cobra.Command{
		Use:   "fetch-ca",
		Short: "Fetch the StackRox CA Configuration to be used with Helm",
		Long:  "Fetch the StackRox CA Configuration to be used with Helm (use this if you are pre-creating secrets rather than using an init bundle)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if outputFile == "" {
				return common.ErrInvalidCommandOption.New("no output file specified with --output (for stdout, specify '-')")
			} else if outputFile == "-" {
				outputFile = ""
			}
			return fetchCAConfig(cliEnvironment, outputFile, flags.Timeout(cmd), flags.RetryTimeout(cmd))
		},
	}
	c.PersistentFlags().StringVar(&outputFile, "output", "", "file to be used for storing the CA config")

	return c
}
