package migratetooperator

import (
	"io"
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/util"
)

type command struct {
	env       environment.Environment
	fromDir   string
	namespace string
	output    string
}

// Command defines the migrate-to-operator command.
func Command(cliEnvironment environment.Environment) *cobra.Command {
	cmd := &command{env: cliEnvironment}
	c := &cobra.Command{
		Use:   "migrate-to-operator",
		Short: "Generate a Central custom resource from existing roxctl-generated manifests",
		Long: `Inspects an existing StackRox Central deployment (from a directory of generated
manifests or a live cluster) and produces a Central custom resource YAML that
preserves the detected configuration, allowing the StackRox operator to
seamlessly take over management of the deployment.`,
		RunE: util.RunENoArgs(func(c *cobra.Command) error {
			return cmd.run()
		}),
	}
	c.Flags().StringVar(&cmd.fromDir, "from-dir", "", "Path to directory containing roxctl-generated manifests")
	c.Flags().StringVarP(&cmd.namespace, "namespace", "n", "", "Kubernetes namespace of the running Central deployment")
	c.Flags().StringVarP(&cmd.output, "output", "o", "", "Path to write the generated Central CR YAML (default: stdout)")
	c.MarkFlagsMutuallyExclusive("from-dir", "namespace")
	return c
}

func (cmd *command) run() error {
	src, err := cmd.createSource()
	if err != nil {
		return err
	}

	config, err := detect(src)
	if err != nil {
		return errors.Wrap(err, "detecting configuration")
	}

	cr := generateCR(config)

	out, err := marshalCR(cr)
	if err != nil {
		return errors.Wrap(err, "marshalling Central CR")
	}

	var w io.Writer = os.Stdout
	if cmd.output != "" {
		f, err := os.Create(cmd.output)
		if err != nil {
			return errors.Wrap(err, "creating output file")
		}
		defer f.Close()
		w = f
	}

	if _, err := w.Write(out); err != nil {
		return errors.Wrap(err, "writing output")
	}
	return nil
}

func (cmd *command) createSource() (source, error) {
	switch {
	case cmd.fromDir != "":
		return newDirSource(cmd.fromDir)
	case cmd.namespace != "":
		return newClusterSource(cmd.namespace)
	default:
		return nil, errors.New("either --from-dir or --namespace must be specified")
	}
}
