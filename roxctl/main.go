package main

import (
	"os"

	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/roxctl/common"
	"github.com/stackrox/rox/roxctl/maincommand"

	// Make sure devbuild setting is registered.
	_ "github.com/stackrox/rox/pkg/devbuild"
)

func main() {
	c := maincommand.Command()

	// This is a workaround. Cobra/pflag takes care of presenting flag usage information
	// to the user including the respective flag default values.
	//
	// But, as an exception, showing the default value for a flag is skipped in pflag if
	// that value is the zero value for a certain standard type.
	//
	// In our case this caused the unintended behaviour of not showing the default values
	// for our boolean flags which default to `false`.
	//
	// Until we have a better solution (e.g. way to control this behaviour in upstream pflag)
	// we simply add the usage information "(default false)" to our affected boolean flags.
	AddMissingDefaultsToFlagUsage(c)

	common.PatchPersistentPreRunHooks(c)

	clientconn.SetUserAgent(clientconn.Roxctl)

	if err := c.Execute(); err != nil {
		os.Exit(1)
	}
}
