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

// CentralVersionHeader is the gRPC response metadata key carrying the
// Central version. Set by Central for authenticated requests so that
// clients (e.g. roxctl) can detect version skew without an extra RPC.
const CentralVersionHeader = "rh-central-version"
