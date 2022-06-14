package restore

import (
	"io"
	"net/http"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/stackrox/pkg/httputil"
	"github.com/stackrox/stackrox/pkg/utils"
	"github.com/stackrox/stackrox/roxctl/central/db/transfer"
)

func (cmd *centralDbRestoreCommand) restoreV1(file *os.File, deadline time.Time) error {
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return errors.Wrap(err, "could not rewind to beginning of file")
	}

	client, err := cmd.env.HTTPClient(0)
	if err != nil {
		return err
	}

	req, err := client.NewReq(http.MethodPost, "/db/restore", file)
	if err != nil {
		return err
	}

	resp, err := transfer.ViaHTTP(req, client, deadline, idleTimeout)
	if err != nil {
		return err
	}
	defer utils.IgnoreError(resp.Body.Close)

	return httputil.ResponseToError(resp)
}
