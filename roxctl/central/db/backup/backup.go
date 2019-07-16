package backup

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/roxctl/central/db/transfer"
	"github.com/stackrox/rox/roxctl/common"
	"github.com/stackrox/rox/roxctl/common/flags"
)

const (
	idleTimeout = 5 * time.Minute
)

// Command defines the db backup command
func Command() *cobra.Command {
	c := &cobra.Command{
		Use:          "backup",
		Short:        "Save a snapshot of the DB as a backup.",
		Long:         "Save a snapshot of the DB as a backup.",
		SilenceUsage: true,
		RunE: func(c *cobra.Command, _ []string) error {
			return getBackup(flags.Timeout(c))
		},
	}

	return c
}

func parseFilenameFromHeader(header http.Header) (string, error) {
	data := header.Get("Content-Disposition")
	if data == "" {
		return data, fmt.Errorf("could not parse filename from header: %+v", header)
	}
	data = strings.TrimPrefix(data, "attachment; filename=")
	return strings.Trim(data, `"`), nil
}

func parseContentLengthFromHeader(header http.Header) (int64, error) {
	data := header.Get("Content-Length")
	if data == "" {
		return 0, nil
	}
	return strconv.ParseInt(data, 10, 64)
}

func getBackup(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	req, err := http.NewRequest("GET", common.GetURL("/db/backup"), nil)
	if err != nil {
		return err
	}
	common.AddAuthToRequest(req)

	reqCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req = req.WithContext(reqCtx)

	client := common.GetHTTPClient(0)

	// Cancel the context if no headers have been received after `timeout`.
	t := time.AfterFunc(timeout, cancel)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	if !t.Stop() {
		// The context will be canceled so we also can't do any reads.
		return errors.New("server took too long to send headers")
	}

	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusUnauthorized, http.StatusForbidden:
		return fmt.Errorf("Invalid credentials. Please add/fix your credentials")
	default:
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("Error with status code %d when trying to get a backup", resp.StatusCode)
		}
		return fmt.Errorf("Error with status code %d when trying to get a backup. Response body: %s", resp.StatusCode, string(body))
	}

	filename, err := parseFilenameFromHeader(resp.Header)
	if err != nil {
		return err
	}
	totalSize, err := parseContentLengthFromHeader(resp.Header)
	if err != nil {
		return err
	}

	file, err := os.Create(filename)
	if err != nil {
		return errors.Wrapf(err, "Could not create backup file %q", filename)
	}
	defer utils.IgnoreError(file.Close)

	if err := transfer.Copy(reqCtx, cancel, filename, totalSize, resp.Body, file, deadline, idleTimeout); err != nil {
		return err
	}

	fmt.Printf("Wrote DB backup file to %q\n", filename)
	return file.Close()
}
