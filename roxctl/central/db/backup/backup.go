package backup

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/roxctl/common"
)

// Command defines the db backup command
func Command() *cobra.Command {
	c := &cobra.Command{
		Use:          "backup",
		Short:        "Save a snapshot of the DB as a backup.",
		Long:         "Save a snapshot of the DB as a backup.",
		SilenceUsage: true,
		RunE: func(c *cobra.Command, _ []string) error {
			return getBackup()
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

func getBackup() error {
	req, err := http.NewRequest("GET", common.GetURL("/db/backup"), nil)
	if err != nil {
		return err
	}
	common.AddAuthToRequest(req)

	client := common.GetHTTPClient()
	resp, err := client.Do(req)
	if err != nil {
		return err
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
	file, err := os.Create(filename)
	if err != nil {
		return errors.Wrapf(err, "Could not create backup file %q", filename)
	}

	if _, err := io.Copy(file, resp.Body); err != nil {
		return err
	}
	fmt.Printf("Wrote DB backup file to %q\n", filename)
	return nil
}
