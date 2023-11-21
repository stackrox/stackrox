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
	scannerUploadDBAPIPath = "/api/extensions/scannerdefinitions"
)

type scannerUploadDBCommand struct {
	// Properties that are bound to cobra flags.
	filename string
	timeout  time.Duration

	// Properties that are injected or constructed.
	env environment.Environment
}

func (cmd *scannerUploadDBCommand) construct(c *cobra.Command) {
	cmd.timeout = flags.Timeout(c)
}

func (cmd *scannerUploadDBCommand) uploadDB() error {
	file, err := os.Open(cmd.filename)
	if err != nil {
		return errors.Wrap(err, "could not open file")
	}
	defer utils.IgnoreError(file.Close)

	client, err := cmd.env.HTTPClient(cmd.timeout)
	if err != nil {
		return errors.Wrap(err, "creating HTTP client")
	}
	resp, err := client.DoReqAndVerifyStatusCode(scannerUploadDBAPIPath, http.MethodPost, http.StatusOK, file)
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
	scannerUploadDBCmd := &scannerUploadDBCommand{env: cliEnvironment}

	c := &cobra.Command{
		Use:   "upload-db",
		Short: "Upload a vulnerability database for the StackRox Scanner.",
		Args:  cobra.NoArgs,
		RunE: func(c *cobra.Command, args []string) error {
			scannerUploadDBCmd.construct(c)

			return scannerUploadDBCmd.uploadDB()
		},
	}

	c.Flags().StringVar(&scannerUploadDBCmd.filename, "scanner-db-file", "", "File containing the dumped Scanner definitions DB")
	flags.AddTimeoutWithDefault(c, 10*time.Minute)
	utils.Must(c.MarkFlagRequired("scanner-db-file"))

	return c
}
