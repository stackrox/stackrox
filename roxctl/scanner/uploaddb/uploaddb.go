package uploaddb

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/roxctl/common"
	"github.com/stackrox/rox/roxctl/common/flags"
)

// Command represents the command.
func Command() *cobra.Command {
	var filename string

	c := &cobra.Command{
		Use:   "upload-db",
		Short: "upload-db uploads a vulnerability database for the StackRox Scanner to use.",
		Long:  "upload-db uploads a vulnerability database for the StackRox Scanner to use.",
		RunE: func(c *cobra.Command, _ []string) error {
			file, err := os.Open(filename)
			if err != nil {
				return errors.Wrap(err, "Could not open file")
			}
			defer utils.IgnoreError(file.Close)

			resp, err := common.DoHTTPRequestAndCheck200("/api/extensions/scannerdefinitions", flags.Timeout(c), "POST", file)
			if err != nil {
				return err
			}
			defer utils.IgnoreError(resp.Body.Close)
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return errors.Wrap(err, "failed to read body")
			}
			fmt.Println(string(body))
			return nil
		},
	}

	c.Flags().StringVar(&filename, "scanner-db-file", "", "file containing the dumped Scanner definitions DB")
	utils.Must(c.MarkFlagRequired("scanner-db-file"))
	return c
}
