package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/pkg/errors"
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
	if len(args) == 0 {
		return errors.New("usage: ci-select validate | resolve --files-file PATH | gha [JOB]")
	}
	command := args[0]
	if command == "gha" {
		if len(args) > 2 {
			return errors.New("gha takes at most one job name")
		}
		return gha(ctx, strings.Join(args[1:], ""), output, diagnostic)
	}
	if command != "validate" && command != "resolve" {
		return fmt.Errorf("unknown command %q", command)
	}
	flags := flag.NewFlagSet(command, flag.ContinueOnError)
	flags.SetOutput(diagnostic)
	rules := flags.String("rules", "ci/test-selection-rules.yaml", "YAML rules")
	var filesPath string
	if command == "resolve" {
		flags.StringVar(&filesPath, "files-file", "", "Changed paths, one per line")
	}
	if err := flags.Parse(args[1:]); err != nil {
		return errors.Wrap(err, "parse flags")
	}
	if flags.NArg() != 0 {
		return errors.New("unexpected positional arguments")
	}
	m, err := loadMapping(*rules)
	if err != nil {
		return err
	}
	if command == "validate" {
		_, err := fmt.Fprintln(output, "Validation passed")
		return errors.Wrap(err, "print validation")
	}
	if filesPath == "" {
		return errors.New("resolve needs --files-file")
	}
	data, err := os.ReadFile(filesPath)
	if err != nil {
		return errors.Wrap(err, "read changed paths")
	}
	var files []string
	for line := range strings.SplitSeq(string(data), "\n") {
		if line != "" {
			files = append(files, line)
		}
	}
	d := resolve(files, m)
	fmt.Fprint(diagnostic, d.log)
	_, err = fmt.Fprint(output, d.String())
	return errors.Wrap(err, "print decision")
}
