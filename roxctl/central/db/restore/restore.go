package restore

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/spf13/cobra"
	"github.com/stackrox/rox/roxctl/common"
)

// Command defines the db backup command
func Command() *cobra.Command {
	var file string
	c := &cobra.Command{
		Use:   "restore",
		Short: "Restore the Central DB from a local file.",
		Long:  "Restore the Central DB from a local file.",
		RunE: func(c *cobra.Command, _ []string) error {
			if file == "" {
				return fmt.Errorf("file to restore from must be specified")
			}
			return restore(file)
		},
	}

	c.Flags().StringVar(&file, "file", "", "file to restore the DB from")
	return c
}

func restore(file string) error {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", common.GetURL("/db/restore"), bytes.NewBuffer(data))
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
		fmt.Println("Successfully restored DB")
		return nil
	case http.StatusUnauthorized, http.StatusForbidden:
		return fmt.Errorf("Token is not authorized to restore DB")
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	return fmt.Errorf("Received response code %d, but expected 200. Response body: %s", resp.StatusCode, string(body))
}
