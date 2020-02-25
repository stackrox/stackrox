package cert

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/roxctl/common"
	"github.com/stackrox/rox/roxctl/common/flags"
)

// Command defines the cert command tree
func Command() *cobra.Command {
	var filename string
	cmd := &cobra.Command{
		Use:   "cert",
		Short: "Download Central's TLS certificate",
		Long:  "Downloads the public certificate used by Central",
		RunE: func(_ *cobra.Command, _ []string) error {
			return certs(filename)
		},
	}

	cmd.Flags().StringVar(&filename, "output", "-", "Filename to output PEM certificate to (default: - for stdout)")
	return cmd
}

func certs(filename string) error {
	// Parse out the endpoint and server name for connecting to.
	endpoint, serverName, err := common.ConnectNames()
	if err != nil {
		return err
	}

	// Connect to the given server. We're not expecting the endpoint be
	// trusted, but force the user to use insecure mode if needed.
	config := tls.Config{
		InsecureSkipVerify: skipTLSValidation(),
		ServerName:         serverName,
	}
	conn, err := tls.Dial("tcp", endpoint, &config)
	if err != nil {
		return err
	}
	defer utils.IgnoreError(conn.Close)

	// Verify that at least 1 certificate was obtained from the connection.
	certs := conn.ConnectionState().PeerCertificates
	if len(certs) == 0 {
		return errors.New("server returned no certificates")
	}

	// "File" to output PEM certificate to.
	var handle io.WriteCloser

	switch filename {
	case "-":
		// Default to STDOUT.
		handle = os.Stdout
	default:
		// Open the given filename.
		handle, err = os.Create(filename)
		if err != nil {
			return err
		}
	}

	// Print out information about the leaf cert to STDERR.
	writeCertInfo(os.Stderr, certs[0])

	// Write out the leaf cert in PEM format.
	if err := writeCertPEM(handle, certs[0]); err != nil {
		return err
	}
	return handle.Close()
}

func skipTLSValidation() bool {
	if value := flags.SkipTLSValidation(); value != nil {
		return *value
	}
	return false
}

func writeCertPEM(writer io.Writer, cert *x509.Certificate) error {
	var pemkey = &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert.Raw,
	}
	if err := pem.Encode(writer, pemkey); err != nil {
		return err
	}
	return nil
}

func writeCertInfo(writer io.Writer, cert *x509.Certificate) {
	fmt.Fprintf(writer, "Issuer:  %v\n", cert.Issuer)
	fmt.Fprintf(writer, "Subject: %v\n", cert.Subject)
	fmt.Fprintf(writer, "Not valid before: %v\n", cert.NotBefore)
	fmt.Fprintf(writer, "Not valid after:  %v\n", cert.NotAfter)
}
