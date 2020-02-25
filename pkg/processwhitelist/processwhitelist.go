package processwhitelist

import "github.com/stackrox/rox/generated/storage"

// WhitelistItemFromProcess returns what we whitelist for a given process.
// It exists to make sure that we're using the same thing in every place (name vs execfilepath).
func WhitelistItemFromProcess(process *storage.ProcessIndicator) string {
	return process.GetSignal().GetExecFilePath()
}
