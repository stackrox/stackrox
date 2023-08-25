package common

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// PatchPersistentPreRunHooks patches the tree of commands beginning from the provided command in order to
// support chaining of persistent pre-run hooks.
//
// As of now (July 2020) Cobra does not support hook chaining. This means that a subcommand defining
// its own PersistentPreRun(E) hook effectively overwrites the PersistentPreRun(E) hook from its parent
// command. Therefore a command cannot simply implement sanity checks (e.g. for deprecated flags) and expect
// these checks to be run in the presence of subcommands (which *also* might need to implement their own
// sanity checks).
//
// There are multiple workarounds possible to mitigate this. This workaround (see below) modifies the
// PersistentPreRun(E) hooks of subcommands such that in addition to the subcommand's hook the parent's
// PersistentPreRun(E) hook is also run.
//
// This can be removed once Cobra supports this kind of chaining of pre-run hooks out of the box.
func PatchPersistentPreRunHooks(c *cobra.Command) {
	for _, child := range c.Commands() {
		thisHook := child.PersistentPreRun
		thisHookE := child.PersistentPreRunE
		child.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
			// Call parent's hook first.
			if c.PersistentPreRunE != nil {
				err := c.PersistentPreRunE(c, args)
				if err != nil {
					return errors.WithStack(err)
				}
			} else if c.PersistentPreRun != nil {
				c.PersistentPreRun(c, args)
			}

			// then the one for this child.
			if thisHookE != nil {
				err := thisHookE(cmd, args)
				if err != nil {
					return errors.WithStack(err)
				}
			} else if thisHook != nil {
				thisHook(cmd, args)
			}

			return nil
		}

		PatchPersistentPreRunHooks(child)
	}
}
