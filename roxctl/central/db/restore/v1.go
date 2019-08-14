package restore

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/roxctl/central/db/transfer"
	"github.com/stackrox/rox/roxctl/common"
	"github.com/stackrox/rox/roxctl/common/flags"
)

func restoreV1(file *os.File, deadline time.Time) error {
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return errors.Wrap(err, "could not rewind to beginning of file")
	}

	req, err := http.NewRequest("POST", common.GetURL("/db/restore"), file)
	if err != nil {
		return err
	}
	common.AddAuthToRequest(req)

	client := common.GetHTTPClient(0)

	resp, err := transfer.ViaHTTP(req, client, deadline, idleTimeout)
	if err != nil {
		return err
	}
	defer utils.IgnoreError(resp.Body.Close)

	return httputil.ResponseToError(resp)
}

// V1Command defines the legacy db restore command
func V1Command() *cobra.Command {
	var file string
	c := &cobra.Command{
		Use:   "restore",
		Short: "Restore the Central DB from a local file.",
		Long:  "Restore the Central DB from a local file.",
		RunE: func(c *cobra.Command, _ []string) error {
			if file == "" {
				return fmt.Errorf("file to restore from must be specified")
			}
			return restore(file, flags.Timeout(c), restoreV1)
		},
	}

	c.Flags().StringVar(&file, "file", "", "file to restore the DB from")
	return c
}
