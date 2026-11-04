package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/ci/select/internal/resolver"
	"github.com/stackrox/rox/ci/select/internal/rules"
)

//nolint:forbidigo // The entry point supplies standard streams to the injectable command runner.
func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	if err := run(ctx, os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "ci-select:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, output, diagnostic io.Writer) error {
	root := &cobra.Command{
		Use:               "ci-select",
		Short:             "Select CI tests from changed paths",
		Args:              cobra.NoArgs,
		SilenceUsage:      true,
		SilenceErrors:     true,
		CompletionOptions: cobra.CompletionOptions{DisableDefaultCmd: true},
		RunE: func(_ *cobra.Command, _ []string) error {
			return errors.New("a subcommand is required (resolve)")
		},
	}
	root.AddCommand(newResolveCommand())
	if args == nil {
		args = []string{}
	}
	root.SetArgs(args)
	root.SetOut(output)
	root.SetErr(diagnostic)
	return errors.Wrap(root.ExecuteContext(ctx), "execute command")
}

func newResolveCommand() *cobra.Command {
	var (
		rulesPath string
		filesPath string
	)
	command := &cobra.Command{
		Use:   "resolve",
		Short: "Resolve test decisions from changed paths",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			data, err := os.ReadFile(rulesPath)
			if err != nil {
				return errors.Wrap(err, "read rules")
			}
			m, err := rules.Parse(data)
			if err != nil {
				return err
			}
			for _, rule := range m.Rules {
				if rule == nil {
					continue
				}
				if err := rule.Compile(); err != nil {
					return errors.Wrap(err, "compile patterns")
				}
			}
			if filesPath == "" {
				return errors.New("resolve needs --files-file")
			}
			data, err = os.ReadFile(filesPath)
			if err != nil {
				return errors.Wrap(err, "read changed paths")
			}
			var files []string
			for line := range strings.SplitSeq(string(data), "\n") {
				if line != "" {
					files = append(files, line)
				}
			}
			d := resolver.Resolve(files, m)
			fmt.Fprint(cmd.ErrOrStderr(), d.Log)
			_, err = fmt.Fprint(cmd.OutOrStdout(), "\n-- decisions --\n"+d.String())
			return errors.Wrap(err, "print decision")
		},
	}
	command.Flags().StringVar(&rulesPath, "rules", "ci/test-selection-rules.yaml", "YAML rules")
	command.Flags().StringVar(&filesPath, "files-file", "", "Changed paths, one per line")
	return command
}
