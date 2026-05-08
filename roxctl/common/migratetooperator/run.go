package migratetooperator

import (
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/yaml"

	migrate "github.com/stackrox/rox/pkg/migratetooperator"
	"github.com/stackrox/rox/roxctl/common/environment"
)

// Command holds the common state for migrate-to-operator subcommands.
type Command struct {
	Env       environment.Environment
	FromDir   string
	Namespace string
	Output    string
}

// AddFlags registers the common flags on the given cobra command.
func (cmd *Command) AddFlags(c *cobra.Command) {
	c.Flags().StringVar(&cmd.FromDir, "from-dir", "", "Path to directory containing generated manifests.")
	c.Flags().StringVarP(&cmd.Namespace, "namespace", "n", "", "Kubernetes namespace of the running deployment.")
	c.Flags().StringVarP(&cmd.Output, "output", "o", "", "Path to write the generated CR YAML (default: stdout).")
	c.MarkFlagsOneRequired("from-dir", "namespace")
	c.MarkFlagsMutuallyExclusive("from-dir", "namespace")
}

// Run creates a Source from the given flags, calls transform, emits warnings,
// marshals the result to YAML, and writes it to output (or stdout).
func Run[T runtime.Object](cmd *Command, transform func(migrate.Source) (T, []string, error)) (retErr error) {
	src, err := createSource(cmd.FromDir, cmd.Namespace)
	if err != nil {
		return err
	}

	cr, warnings, err := transform(src)
	if err != nil {
		return errors.Wrap(err, "detecting configuration")
	}

	for _, w := range warnings {
		cmd.Env.Logger().WarnfLn(w)
	}

	out, err := yaml.Marshal(cr)
	if err != nil {
		return errors.Wrap(err, "marshalling custom resource")
	}

	w := cmd.Env.InputOutput().Out()
	if cmd.Output != "" {
		f, err := os.Create(cmd.Output)
		if err != nil {
			return errors.Wrap(err, "creating output file")
		}
		defer func() {
			if cerr := f.Close(); cerr != nil && retErr == nil {
				retErr = errors.Wrap(cerr, "closing output file")
			}
		}()
		w = f
	}

	if _, err := w.Write(out); err != nil {
		return errors.Wrap(err, "writing output")
	}
	return nil
}

func createSource(fromDir, namespace string) (migrate.Source, error) {
	switch {
	case fromDir != "":
		src, err := migrate.NewDirSource(fromDir)
		if err != nil {
			return nil, errors.Wrap(err, "reading manifests from directory")
		}
		return src, nil
	case namespace != "":
		src, err := migrate.NewClusterSource(namespace)
		if err != nil {
			return nil, errors.Wrap(err, "connecting to cluster")
		}
		return src, nil
	default:
		return nil, errors.New("either --from-dir or --namespace must be specified")
	}
}
