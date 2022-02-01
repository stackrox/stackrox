package initbundles

import (
	"context"
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	v1 "github.com/stackrox/rox/generated/api/v1"
	pkgCommon "github.com/stackrox/rox/pkg/roxctl/common"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/roxctl/common"
)

func fetchCAConfig(outputFile string) error {
	ctx, cancel := context.WithTimeout(pkgCommon.Context(), contextTimeout)
	defer cancel()

	conn, err := common.GetGRPCConnection()
	if err != nil {
		return err
	}
	defer utils.IgnoreError(conn.Close)
	svc := v1.NewClusterInitServiceClient(conn)

	bundleOutput := os.Stdout
	if outputFile != "" {
		bundleOutput, err = os.OpenFile(outputFile, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
		if err != nil {
			return errors.Wrap(err, "opening output file for writing CA config")
		}
		defer func() {
			if bundleOutput != nil {
				_ = bundleOutput.Close()
				utils.Should(os.Remove(outputFile))
			}
		}()
	}

	resp, err := svc.GetCAConfig(ctx, &v1.Empty{})
	if err != nil {
		return errors.Wrap(err, "fetching CA config")
	}

	_, err = bundleOutput.Write(resp.GetHelmValuesBundle())
	if err != nil {
		return errors.Wrap(err, "writing init bundle")
	}
	if bundleOutput != os.Stdout {
		fmt.Fprintf(os.Stderr, "The CA configuration has been written to file %q.\n", outputFile)
		if err := bundleOutput.Close(); err != nil {
			return errors.Wrap(err, "closing output file for CA config")
		}
		bundleOutput = nil
	}

	return nil
}

func fetchCACommand() *cobra.Command {
	var outputFile string

	c := &cobra.Command{
		Use:  "fetch-ca",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if outputFile == "" {
				return errors.New("No output file specified with --output (for stdout, specify '-')")
			} else if outputFile == "-" {
				outputFile = ""
			}
			return fetchCAConfig(outputFile)
		},
	}
	c.PersistentFlags().StringVar(&outputFile, "output", "", "file to be used for storing the CA config")

	return c
}
