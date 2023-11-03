package initbundles

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

type output struct {
	format   func(request *v1.InitBundleGenResponse) []byte
	filename string
}

func generateInitBundle(cliEnvironment environment.Environment, name string,
	outputs []output, timeout time.Duration, retryTimeout time.Duration,
) error {
	ctx, cancel := context.WithTimeout(pkgCommon.Context(), timeout)
	defer cancel()

	conn, err := cliEnvironment.GRPCConnection(common.WithRetryTimeout(retryTimeout))
	if err != nil {
		return err
	}
	defer utils.IgnoreError(conn.Close)
	svc := v1.NewClusterInitServiceClient(conn)

	files := make([]*os.File, 0, len(outputs))
	defer func() {
		for _, f := range files {
			if f != nil && f != os.Stdout { //nolint:forbidigo // TODO(ROX-13473)
				name := f.Name()
				_ = f.Close()
				utils.Should(os.Remove(name))
			}
		}
	}()

	// First try to open all files. Since creating a bundle has side effects, let's not attempt to do so
	// before we have high confidence that the writing will succeed.
	for _, out := range outputs {
		outFile := os.Stdout //nolint:forbidigo // TODO(ROX-13473)
		if out.filename != "" {
			outFile, err = os.OpenFile(out.filename, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
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

	cliEnvironment.Logger().InfofLn(`Successfully generated new init bundle.

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
		if outFile != os.Stdout { //nolint:forbidigo // TODO(ROX-13473)
			cliEnvironment.Logger().InfofLn("The newly generated init bundle has been written to file %q.", outFile.Name())
			if err := outFile.Close(); err != nil {
				return errors.Wrapf(err, "closing output file %q", outFile.Name())
			}
		}
		files[i] = nil
	}

	cliEnvironment.Logger().InfofLn("The init bundle needs to be stored securely, since it contains secrets.")
	cliEnvironment.Logger().InfofLn("It is not possible to retrieve previously generated init bundles.")
	return nil
}

// generateCommand implements the command for generating new init bundles.
func generateCommand(cliEnvironment environment.Environment) *cobra.Command {
	var outputFile string
	var secretsOutputFile string

	var outputs []output

	c := &cobra.Command{
		Use:   "generate <init bundle name>",
		Short: "Generate a new cluster init bundle",
		Long:  "Generate a new init bundle for bootstrapping a new StackRox secured cluster",
		Args:  common.ExactArgsWithCustomErrMessage(1, "No name for the init bundle specified"),
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
				return common.ErrInvalidCommandOption.New("No output files specified with --output or --output-secrets (for stdout, specify '-')")
			}
			return generateInitBundle(cliEnvironment, name, outputs, flags.Timeout(cmd), flags.RetryTimeout(cmd))
		},
	}
	c.PersistentFlags().StringVar(&outputFile, "output", "", "file to be used for storing the newly generated init bundle in Helm configuration form (- for stdout)")
	c.PersistentFlags().StringVar(&secretsOutputFile, "output-secrets", "", "file to be used for storing the newly generated init bundle in Kubernetes secrets form (- for stdout)")

	return c
}
