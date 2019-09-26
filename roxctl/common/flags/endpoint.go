package flags

import (
	"github.com/spf13/cobra"
)

var (
	endpoint   string
	serverName string
	directGRPC bool
)

// AddEndpoint adds the endpoint flag to the base command.
func AddEndpoint(c *cobra.Command) {
	c.PersistentFlags().StringVarP(&endpoint, "endpoint", "e", "localhost:8443", "endpoint for service to contact")
}

// AddServerName adds the server-name flag to the base command.
func AddServerName(c *cobra.Command) {
	c.PersistentFlags().StringVarP(&serverName, "server-name", "s", "", "TLS ServerName to use for SNI (if empty, derived from endpoint)")
}

// AddDirectGRPC adds the direct-grpc flag to the command.
func AddDirectGRPC(c *cobra.Command) {
	c.PersistentFlags().BoolVar(&directGRPC, "direct-grpc", false, "Use direct gRPC (advanced; only use if you encounter connection issues)")
}

// Endpoint returns the set endpoint.
func Endpoint() string {
	return endpoint
}

// ServerName returns the specified ServerName.
func ServerName() string {
	return serverName
}

// UseDirectGRPC returns whether to use gRPC directly, i.e., without a proxy.
func UseDirectGRPC() bool {
	return directGRPC
}
