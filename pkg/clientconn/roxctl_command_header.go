package clientconn

// RoxctlCommandHeader is the HTTP header name with the reconstructed
// roxctl command as the value.
const RoxctlCommandHeader = "Rh-Roxctl-Command"

// RoxctlCommandIndexHeader is the name of the HTTP header, which value
// represents the sequential number of the roxctl API call executed for the
// CLI command, provided in RoxctlCommandHeader.
const RoxctlCommandIndexHeader = "Rh-Roxctl-Command-Index"

// ExecutionEnvironment is the HTTP header name with the custom execution
// environment string. Can be supplied by roxctl, or any other API client.
const ExecutionEnvironment = "Rh-Execution-Environment"
