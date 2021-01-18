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

func generateInitBundle(name string, outputFile string) error {
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
		bundleOutput, err = os.OpenFile(outputFile, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			return errors.Wrap(err, "opening output file for writing init bundle")
		}
		defer func() {
			if outputFile != "" {
				utils.Should(os.Remove(outputFile))
			}
		}()
	}

	req := v1.InitBundleGenRequest{Name: name}
	resp, err := svc.GenerateInitBundle(ctx, &req)
	if err != nil {
		return errors.Wrap(err, "generating new init bundle")
	}

	meta := resp.GetMeta()

	fmt.Fprintf(os.Stderr, `Successfully generated new init bundle.

  ID:         %s
  Name:       %q
  Expires at: %v

`, meta.GetId(), meta.GetName(), meta.GetExpiresAt())

	_, err = bundleOutput.Write(resp.GetHelmValuesBundle())
	if err != nil {
		return errors.Wrap(err, "writing init bundle")
	}
	if bundleOutput != os.Stdout {
		fmt.Fprintf(os.Stderr, "The newly generated init bundle has been written to file %q.\n", outputFile)
		if err := bundleOutput.Close(); err != nil {
			return errors.Wrap(err, "closing output file for init bundle")
		}
		outputFile = "" // Make sure that file will not be deleted by deferred cleanup handler.
	}

	fmt.Fprintln(os.Stderr, `
Note: The init bundle needs to be stored securely, since it contains secrets.
      It is not possible to retrieve previously generated init bundles.`)
	return nil
}

// generateCommand implements the command for generating new init bundles.
func generateCommand() *cobra.Command {
	var outputFile string

	c := &cobra.Command{
		Use:  "generate <init bundle name>",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			if outputFile == "" {
				return errors.New("No output file specified with --output (for stdout, specify '-')")
			} else if outputFile == "-" {
				outputFile = ""
			}
			return generateInitBundle(name, outputFile)
		},
	}
	c.PersistentFlags().StringVar(&outputFile, "output", "", "file to be used for storing the newly generated init bundle")

	return c
}
