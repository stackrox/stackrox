package cert

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"io"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/ioutils"
	pkgCommon "github.com/stackrox/rox/pkg/roxctl/common"
	"github.com/stackrox/rox/pkg/tlsutils"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/flags"
	"github.com/stackrox/rox/roxctl/common/logger"
	"github.com/stackrox/rox/roxctl/common/util"
)

type centralCertCommand struct {
	// Properties that are bound to cobra flags.
	filename string

	// Properties that are injected or constructed.
	env     environment.Environment
	timeout time.Duration
}

// Command defines the cert command tree
func Command(cliEnvironment environment.Environment) *cobra.Command {
	centralCertCommand := &centralCertCommand{env: cliEnvironment}
	cbr := &cobra.Command{
		Use:   "cert",
		Short: "Download certificate chain for the Central service.",
		Long:  "Download certificate chain for the Central service or its associated ingress or load balancer, if one exists.",
		RunE: util.RunENoArgs(func(cmd *cobra.Command) error {
			if err := centralCertCommand.construct(cmd); err != nil {
				return err
			}
			return centralCertCommand.certs()
		}),
	}

	cbr.Flags().StringVar(&centralCertCommand.filename, "output", "-", "Filename to output PEM certificate to; '-' for stdout")
	flags.AddTimeout(cbr)
	flags.AddRetryTimeout(cbr)
	return cbr
}

func (cmd *centralCertCommand) construct(cbr *cobra.Command) error {
	cmd.timeout = flags.Timeout(cbr)
	return nil
}

func (cmd *centralCertCommand) certs() error {
	// Parse out the endpoint and server name for connecting to.
	endpoint, serverName, err := cmd.env.ConnectNames()
	if err != nil {
		return err
	}

	// Connect to the given server. We're not expecting the endpoint be
	// trusted, but force the user to use insecure mode if needed.
	config := tls.Config{
		InsecureSkipVerify: skipTLSValidation(),
		ServerName:         serverName,
	}
	ctx, cancel := context.WithTimeout(pkgCommon.Context(), cmd.timeout)
	defer cancel()
	conn, err := tlsutils.DialContext(ctx, "tcp", endpoint, &config)
	if err != nil {
		return err
	}
	defer utils.IgnoreError(conn.Close)

	// Verify that at least 1 certificate was obtained from the connection.
	certs := conn.ConnectionState().PeerCertificates
	if len(certs) == 0 {
		return errox.NotFound.New("server returned no certificates")
	}

	// "File" to output PEM certificate to.
	var handle io.WriteCloser

	switch cmd.filename {
	case "-":
		// Default to STDOUT.
		handle = ioutils.NopWriteCloser(cmd.env.InputOutput().Out())
	default:
		// Open the given filename.
		handle, err = os.Create(cmd.filename)
		if err != nil {
			return err
		}
	}

	// Print out information about the leaf cert to STDERR.
	writeCertInfo(cmd.env.Logger(), certs[0])

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

func writeCertInfo(logger logger.Logger, cert *x509.Certificate) {
	logger.InfofLn("Issuer: %v", cert.Issuer)
	logger.InfofLn("Issuer:  %v", cert.Issuer)
	logger.InfofLn("Subject: %v", cert.Subject)
	logger.InfofLn("Not valid before: %v", cert.NotBefore)
	logger.InfofLn("Not valid after:  %v", cert.NotAfter)
}
