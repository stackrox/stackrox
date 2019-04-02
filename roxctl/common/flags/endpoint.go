package flags

import (
	"github.com/spf13/cobra"
)

var (
	endpoint   string
	serverName string
)

// AddEndpoint adds the endpoint flag to the base command.
func AddEndpoint(c *cobra.Command) {
	c.PersistentFlags().StringVarP(&endpoint, "endpoint", "e", "localhost:8443", "endpoint for service to contact")
}

// AddServerName adds the server-name flag to the base command.
func AddServerName(c *cobra.Command) {
	c.PersistentFlags().StringVarP(&serverName, "server-name", "s", "", "TLS ServerName to use for SNI (if empty, derived from endpoint)")
}

// Endpoint returns the set endpoint.
func Endpoint() string {
	return endpoint
}

// ServerName returns the specified ServerName.
func ServerName() string {
	return serverName
}
