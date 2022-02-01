package initbundles

import (
	"context"
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	v1 "github.com/stackrox/rox/generated/api/v1"
	pkgCommon "github.com/stackrox/rox/pkg/roxctl/common"
	"github.com/stackrox/rox/pkg/stringutils"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/roxctl/common"
)

type output struct {
	format   func(request *v1.InitBundleGenResponse) []byte
	filename string
}

func generateInitBundle(name string, outputs []output) error {
	ctx, cancel := context.WithTimeout(pkgCommon.Context(), contextTimeout)
	defer cancel()

	conn, err := common.GetGRPCConnection()
	if err != nil {
		return err
	}
	defer utils.IgnoreError(conn.Close)
	svc := v1.NewClusterInitServiceClient(conn)

	files := make([]*os.File, 0, len(outputs))
	defer func() {
		for _, f := range files {
			if f != nil && f != os.Stdout {
				name := f.Name()
				_ = f.Close()
				utils.Should(os.Remove(name))
			}
		}
	}()

	// First try to open all files. Since creating a bundle has side effects, let's not attempt to do so
	// before we have high confidence that the writing will succeed.
	for _, out := range outputs {
		outFile := os.Stdout
		if out.filename != "" {
			outFile, err = os.OpenFile(out.filename, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
			if err != nil {
				return errors.Wrap(err, "opening output file for writing init bundle")
			}
		}
		files = append(files, outFile)
	}

	req := v1.InitBundleGenRequest{Name: name}
	resp, err := svc.GenerateInitBundle(ctx, &req)
	if err != nil {
		return errors.Wrap(err, "generating new init bundle")
	}

	meta := resp.GetMeta()

	fmt.Fprintf(os.Stderr, `Successfully generated new init bundle.

  Name:       %s
  Created at: %v
  Expires at: %v
  Created By: %v
  ID:         %s

`, meta.GetName(), meta.GetCreatedAt(), meta.GetExpiresAt(), getPrettyUser(meta.GetCreatedBy()), meta.GetId())

	for i, out := range outputs {
		outFile := files[i]
		if _, err := outFile.Write(out.format(resp)); err != nil {
			return errors.Wrapf(err, "writing init bundle to %s", stringutils.FirstNonEmpty(out.filename, "<stdout>"))
		}
		if outFile != os.Stdout {
			fmt.Fprintf(os.Stderr, "The newly generated init bundle has been written to file %q.\n", outFile.Name())
			if err := outFile.Close(); err != nil {
				return errors.Wrapf(err, "closing output file %q", outFile.Name())
			}
		}
		files[i] = nil
	}

	fmt.Fprintln(os.Stderr, `
Note: The init bundle needs to be stored securely, since it contains secrets.
      It is not possible to retrieve previously generated init bundles.`)
	return nil
}

// generateCommand implements the command for generating new init bundles.
func generateCommand() *cobra.Command {
	var outputFile string
	var secretsOutputFile string

	var outputs []output

	c := &cobra.Command{
		Use:  "generate <init bundle name>",
		Args: common.ExactArgsWithCustomErrMessage(1, "No name for the init bundle specified"),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			if outputFile != "" {
				if outputFile == "-" {
					outputFile = ""
				}
				outputs = append(outputs, output{
					filename: outputFile,
					format:   (*v1.InitBundleGenResponse).GetHelmValuesBundle,
				})
			}
			if secretsOutputFile != "" {
				if secretsOutputFile == "-" {
					secretsOutputFile = ""
				}
				outputs = append(outputs, output{
					filename: secretsOutputFile,
					format:   (*v1.InitBundleGenResponse).GetKubectlBundle,
				})
			}

			if len(outputs) == 0 {
				return errors.New("No output files specified with --output or --output-secrets (for stdout, specify '-')")
			}
			return generateInitBundle(name, outputs)
		},
	}
	c.PersistentFlags().StringVar(&outputFile, "output", "", "file to be used for storing the newly generated init bundle in Helm configuration form (- for stdout)")
	c.PersistentFlags().StringVar(&secretsOutputFile, "output-secrets", "", "file to be used for storing the newly generated init bundle in Kubernetes secrets form (- for stdout)")

	return c
}
