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
	"github.com/stackrox/rox/roxctl/common"
)

func restoreV1(file *os.File, deadline time.Time) error {
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return errors.Wrap(err, "could not rewind to beginning of file")
	}

	req, err := common.NewHTTPRequestWithAuth(http.MethodPost, "/db/restore", file)
	if err != nil {
		return err
	}

	client, err := common.GetHTTPClient(0)
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
