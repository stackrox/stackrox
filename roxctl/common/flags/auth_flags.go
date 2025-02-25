package flags

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var authFlagSet = func() *pflag.FlagSet {
	fs := pflag.NewFlagSet("auth", pflag.ExitOnError)
	fs.StringVarP(&password, "password", "p", "",
		"Password for basic auth. Alternatively, set the password via the ROX_ADMIN_PASSWORD environment variable")
	passwordChanged = &fs.Lookup("password").Changed

	fs.StringVarP(&apiTokenFile,
		"token-file",
		"",
		"",
		"Use the API token in the provided file to authenticate. "+
			"Alternatively, set the path via the ROX_API_TOKEN_FILE environment variable or "+
			"set the token via the ROX_API_TOKEN environment variable")
	apiTokenFileChanged = &fs.Lookup("token-file").Changed

	return fs
}()

func AddCentralAuthFlags(c *cobra.Command) {
	c.PersistentFlags().AddFlagSet(authFlagSet)
}
