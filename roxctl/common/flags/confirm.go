package flags

import (
	"bufio"
	io2 "io"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/roxctl/common/io"
	"github.com/stackrox/rox/roxctl/common/logger"
)

const (
	forceFlag = "force"
)

// AddForce adds a parameter for bypassing interactive confirmation
func AddForce(c *cobra.Command) {
	c.Flags().BoolP(forceFlag, "f", false, "Proceed without confirmation.")
}

// CheckConfirmation requires that the force argument has been passed or that the user interactively confirms the action
func CheckConfirmation(c *cobra.Command, logger logger.Logger, io io.IO) error {
	f, err := c.Flags().GetBool(forceFlag)
	if err != nil {
		logger.ErrfLn("Error checking value of --force flag: %w", err)
		utils.Should(err)
		f = false
	}
	if f {
		return nil
	}
	logger.PrintfLn("Are you sure? [y/N] ")
	resp, err := ReadUserYesNoConfirmation(io.In())
	if err != nil {
		return errors.Wrap(err, "could not read answer")
	}
	if !resp {
		return errox.NotAuthorized.New("User rejected")
	}
	return nil
}

// ReadUserYesNoConfirmation reads a yes/no confirmation from the user. Empty response defaults to no.
func ReadUserYesNoConfirmation(reader io2.Reader) (bool, error) {
	confirm, err := bufio.NewReader(reader).ReadString('\n')
	if err != nil {
		return false, errors.Wrap(err, "reading user input")
	}
	confirm = strings.ToLower(strings.TrimSpace(confirm))
	if confirm != "y" && confirm != "n" && confirm != "" {
		return false, errors.New("invalid confirmation. Must be 'y' or 'n'")
	}
	return confirm == "y", nil
}
