package flags

import (
	"github.com/spf13/cobra"
)

var (
	endpoint   string
	serverName string
	directGRPC bool

	plaintext bool
	insecure  bool
)

// AddConnectionFlags adds connection-related flags to roxctl.
func AddConnectionFlags(c *cobra.Command) {
	c.PersistentFlags().StringVarP(&endpoint, "endpoint", "e", "localhost:8443", "endpoint for service to contact")
	c.PersistentFlags().StringVarP(&serverName, "server-name", "s", "", "TLS ServerName to use for SNI (if empty, derived from endpoint)")
	c.PersistentFlags().BoolVar(&directGRPC, "direct-grpc", false, "Use direct gRPC (advanced; only use if you encounter connection issues)")

	c.PersistentFlags().BoolVar(&plaintext, "plaintext", false, "Use a plaintext (unencrypted) connection; only works in conjunction with --insecure")
	c.PersistentFlags().BoolVar(&insecure, "insecure", false, "Enable insecure connection options (DANGEROUS; USE WITH CAUTION)")
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

// UsePlaintext returns whether to use a plaintext connection.
func UsePlaintext() bool {
	return plaintext
}

// UseInsecure returns whether to use insecure connection behavior.
func UseInsecure() bool {
	return insecure
}
