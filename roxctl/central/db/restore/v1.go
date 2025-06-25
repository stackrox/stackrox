package restore

import (
	"io"
	"net/http"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/roxctl/central/db/transfer"
)

func (cmd *centralDbRestoreCommand) restoreV1(file *os.File, deadline time.Time) error {
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return errors.Wrap(err, "could not rewind to beginning of file")
	}

	client, err := cmd.env.HTTPClient(0)
	if err != nil {
		return errors.Wrap(err, "creating HTTP client for restore")
	}

	req, err := client.NewReq(http.MethodPost, "/db/restore", file)
	if err != nil {
		return errors.Wrap(err, "creating restore request")
	}

	resp, err := transfer.ViaHTTP(req, client, deadline, idleTimeout)
	if err != nil {
		return errors.Wrap(err, "executing restore request")
	}
	defer utils.IgnoreError(resp.Body.Close)

	return httputil.ResponseToError(resp)
}
