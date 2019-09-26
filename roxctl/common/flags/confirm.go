package flags

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/logging"
)

const (
	forceFlag = "force"
)

var (
	log = logging.LoggerForModule()
)

// AddForce adds a parameter for bypassing interactive confirmation
func AddForce(c *cobra.Command) {
	c.Flags().BoolP(forceFlag, "f", false, "proceed without confirmation")
}

// CheckConfirmation requires that the force argument has been passed or that the user interactively confirms the action
func CheckConfirmation(c *cobra.Command) error {
	f, err := c.Flags().GetBool(forceFlag)
	if err != nil {
		log.Errorf("Error checking value of --force flag: %v", err)
		errorhelpers.PanicOnDevelopment(err)
		f = false
	}
	if f {
		return nil
	}
	fmt.Print("Are you sure? [y/N] ")
	resp, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		return err
	}
	resp = strings.ToLower(strings.TrimSpace(resp))
	if resp != "y" {
		return errors.New("User rejected")
	}
	return nil
}
