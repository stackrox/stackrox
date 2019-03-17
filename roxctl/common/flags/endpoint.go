package flags

import (
	"github.com/spf13/cobra"
)

var (
	endpoint string
)

// AddEndpoint adds the endpoint flag to the base command.
func AddEndpoint(c *cobra.Command) {
	c.PersistentFlags().StringVarP(&endpoint, "endpoint", "e", "localhost:8443", "endpoint for service to contact")
}

// Endpoint returns the set endpoint.
func Endpoint() string {
	return endpoint
}
