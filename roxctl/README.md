# Introduction

`roxctl` is the CLI for Stackrox.

It currently serves as a multi-purpose CLI that handles things such as:

- administrative tasks (e.g. creating backups / restoring them)
- integration within CI for image / workload scanning.
- acting as API client for automation purposes.

The goal of this README is to provide an overview of writing and maintaining commands of `roxctl`.

# Command structure

Commands **must** follow the common name convention: `roxctl <noun> [subcommands...]`.
For more information on why this decision was
made, [an ADR exists explaining it](https://github.com/stackrox/architecture-decision-records/blob/main/stackrox/ADR-0004-roxctl-subcommands-layout.md).

Within the codebase, `roxctl` makes heavy use of [github.com/spf13/cobra](https://pkg.go.dev/github.com/spf13/cobra) as
the CLI library of choice.

Besides `github.com/spf13/cobra`, one integral part how commands are structured is
the [roxctl/common/environment](https://github.com/stackrox/stackrox/tree/master/roxctl/common/environment) package.
The `Environment` interface provides abstractions for generic functionality that may be required by a command:

- Printing output.
- Creating gRPC connections / HTTP client.
- Logging.
- Input / Output abstractions.
- Config store.

Each command used in `roxctl` **must** use the `Environment` interface. It provides a clear structure to all commands,
as well as the possibility to easily mock external dependencies to allow for better and sophisticated unit testing.

A typical command structure within Go should look as follows:

```go
package sample

// Command exposes the *cobra.Command for your specific command.
// It will **always** take a environment.Environment as input.
func Command(cliEnvironment environment.Environment) *cobra.Command {
	sampleCmd := &sampleCmd{env: cliEnvironment}

	cmd := &cobra.Command{
		Use:   "sample",
		Short: "A sample command",
		Long: `A sample command.
This is a sample for a multi-line detailed description for the sample command`,
		Args: cobra.NoArgs,
		// Each command **must** always use `RunE` and return an error.
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := sampleCmd.Construct(cmd); err != nil {
				return err
			}
			return sampleCmd.sample()
		},
	}

	// Each custom flag for a command **must** be bound to the struct holding
	// all relevant properties for a command.
	cmd.Flags().StringVarP(&sampleCmd.name, "name", "n", "", "name to say hello to")

	return cmd
}

// Sample command holds the cli environment, as well as all resolved flag values. This makes unit testing easier.
type sampleCmd struct {
	env environment.Environment

	name string
}

// Construct ensure all flag values are resolved and possibly validated.
// Any additional external dependency that you may require (e.g. a gRPC client)
// may be constructed here as well.
func (s *sampleCmd) Construct(cmd *cobra.Command) error {
	return nil
}

// Sample is an example for any possible business logic your command may execute.
func (s *sampleCmd) sample() error {
	s.env.Logger().InfofLn("Hello world, %s!", s.name)
	return nil
}

```

# Maintaining commands

One of the most important things about maintaining commands for `roxctl` is to ensure changes of behavior have no side
effects.

Since `roxctl` is typically used within CI as well as automation (e.g. scripts) by users, it is sensitive to behavior
changes.

A behavior change may consist of:

- Changing the default value of a flag.
- Removing or renaming a flag.
- Removing or renaming a command.
- Changing non-human readable CLI output (e.g. changing returned JSON).

Since this is a sensitive topic, the following points **must** be considered when maintaining commands:

- Breaking changes **should** be avoided if possible.
- A breaking change **must** have an intrinsic value to it. While this may be vague, here are a couple of examples:
    - Allows to introduce new functionality which provides direct value to a majority of users.
    - Increases stability or usability for a majority of users.
    - Reduces the overall maintenance cost (e.g. removing unused commands, removing support for deprecated
      OpenShift versions)
- Before breaking changes are done, a deprecation notice within the `CHANGELOG.md` **must** be added in advance
  according to our current guidelines for introducing breaking changes.
- When removing or renaming a command or flag, an alias **must** be introduced (see the latter section about this).

## Creating an alias for deprecated values

When changing flags or commands, aliases must be in place to avoid breaking users of said commands.

Currently, two options exist for this:

- Cobra's [alias feature](https://pkg.go.dev/github.com/spf13/cobra#Command).
- Handling the alias in-code.

For Cobra's alias feature, the following limitations currently apply:

- Renaming a command is straight-forward **as long as the command hierarchy does not change**.
- Creating an alias for a flag currently is not supported, only for args.

In case your change falls under the limitations of the cobra alias feature, it's recommended to handle the alias within
the command's code itself.

The below code highlights changes for a command:

```go
package sample

func OldCommand(cliEnvironment environment.Environment) *cobra.Command {
	sampleCmd := &sampleCmd{env: cliEnvironment}

	cmd := &cobra.Command{
		Use:   "old-sample",
		Short: "A sample command",
		Long: `A sample command.
This is a sample for a multi-line detailed description for the sample command.
Also, please use the new-sample command since this one is deprecated.`,
		Hidden: true,
		Args:   cobra.NoArgs,
		// Each command **must** always use `RunE` and return an error.
		RunE: func(cmd *cobra.Command, args []string) error {
			cliEnvironment.Logger().WarnLn("old-sample is deprecated. Please use new-sample.")
			return sampleCmd.RunE(cmd, args)
		},
	}

	cmd.Flags().StringVarP(&sampleCmd.name, "name", "n", "", "name to say hello to")
	return cmd
}

func NewCommand(cliEnvironment environment.Environment) *cobra.Command {
	sampleCmd := &sampleCmd{env: cliEnvironment}

	cmd := &cobra.Command{
		Use:   "new-sample",
		Short: "A sample command",
		Long: `A sample command.
This is a sample for a multi-line detailed description for the sample command`,
		Args: cobra.NoArgs,
		// Each command **must** always use `RunE` and return an error.
		RunE: sampleCmd.RunE,
	}

	cmd.Flags().StringVarP(&sampleCmd.name, "name", "n", "", "name to say hello to")
	return cmd
}

// While the sample command now lives in the same package, it may live in a completely different package.
type sampleCmd struct {
	env environment.Environment

	name string
}

// Aliases can share code and reuse functions, e.g.:
func (s *sampleCmd) RunE(cmd *cobra.Command, args []string) error { ... }
func (s *sampleCmd) Construct(cmd *cobra.Command) error           { ... }
func (s *sampleCmd) sample() error                                { ... }

```
