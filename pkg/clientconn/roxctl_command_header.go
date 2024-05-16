package clientconn

// RoxctlCommandHeader is the HTTP header name with the reconstructed
// roxctl command as the value.
const RoxctlCommandHeader = "Rh-Roxctl-Command"

// RoxctlCommandIndexHeader is the name of the HTTP header, which value
// represents the sequential number of the roxctl API call executed for the
// CLI command, provided in RoxctlCommandHeader.
const RoxctlCommandIndexHeader = "Rh-Roxctl-Command-Index"
