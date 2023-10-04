package uploaddb

import (
	"io"
	"net/http"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/flags"
)

const (
	scannerUploadDbAPIPath = "/api/extensions/scannerdefinitions"
)

type scannerUploadDbCommand struct {
	// Properties that are bound to cobra flags.
	filename string
	timeout  time.Duration

	// Properties that are injected or constructed.
	env environment.Environment
}

func (cmd *scannerUploadDbCommand) construct(c *cobra.Command) {
	cmd.timeout = flags.Timeout(c)
}

func (cmd *scannerUploadDbCommand) uploadDd() error {
	file, err := os.Open(cmd.filename)
	if err != nil {
		return errors.Wrap(err, "could not open file")
	}
	defer utils.IgnoreError(file.Close)

	client, err := cmd.env.HTTPClient(cmd.timeout)
	if err != nil {
		return errors.Wrap(err, "creating HTTP client")
	}
	resp, err := client.DoReqAndVerifyStatusCode(scannerUploadDbAPIPath, http.MethodPost, http.StatusOK, file)
	if err != nil {
		return errors.Wrap(err, "could not connect with scanner definitions API")
	}
	defer utils.IgnoreError(resp.Body.Close)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrap(err, "failed to read body")
	}

	cmd.env.Logger().PrintfLn("%s", string(body))

	return nil
}

// Command represents the command.
func Command(cliEnvironment environment.Environment) *cobra.Command {
	scannerUploadDbCmd := &scannerUploadDbCommand{env: cliEnvironment}

	c := &cobra.Command{
		Use:   "upload-db",
		Short: "Upload a vulnerability database for the StackRox Scanner.",
		Args:  cobra.NoArgs,
		RunE: func(c *cobra.Command, args []string) error {
			scannerUploadDbCmd.construct(c)

			return scannerUploadDbCmd.uploadDd()
		},
	}

	c.Flags().StringVar(&scannerUploadDbCmd.filename, "scanner-db-file", "", "File containing the dumped Scanner definitions DB")
	flags.AddTimeoutWithDefault(c, 10*time.Minute)
	flags.AddRetryTimeoutWithDefault(c, time.Duration(0))
	utils.Must(c.MarkFlagRequired("scanner-db-file"))

	return c
}
