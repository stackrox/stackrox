package flags

import (
	"bufio"
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
	c.Flags().BoolP(forceFlag, "f", false, "proceed without confirmation")
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
	resp, err := bufio.NewReader(io.In()).ReadString('\n')
	if err != nil {
		return errors.Wrap(err, "could not read answer")
	}
	resp = strings.ToLower(strings.TrimSpace(resp))
	if resp != "y" {
		return errox.NotAuthorized.New("User rejected")
	}
	return nil
}
